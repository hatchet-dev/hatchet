// Package sns provides helper functions for verifying and processing Amazon AWS SNS HTTP POST payloads.
// Based on https://github.com/robbiet480/go.sns with updates for Go 1.21+
package sns

import (
	"bytes"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"time"
)

// regional SNS endpoints, e.g. sns.us-east-1.amazonaws.com or sns.cn-north-1.amazonaws.com.cn
var hostPattern = regexp.MustCompile(`^sns\.[a-z]{2,}(-[a-z]+)+-[0-9]+\.amazonaws\.com(\.cn)?$`)

const (
	maxResponseBytes = 1 << 20
	maxTimestampSkew = time.Hour
)

var httpClient = &http.Client{Timeout: 10 * time.Second}

// Payload contains a single POST from SNS
type Payload struct {
	Message          string `json:"Message"`
	MessageId        string `json:"MessageId"`
	Signature        string `json:"Signature"`
	SignatureVersion string `json:"SignatureVersion"`
	SigningCertURL   string `json:"SigningCertURL"`
	SubscribeURL     string `json:"SubscribeURL"`
	Subject          string `json:"Subject"`
	Timestamp        string `json:"Timestamp"`
	Token            string `json:"Token"`
	TopicArn         string `json:"TopicArn"`
	Type             string `json:"Type"`
	UnsubscribeURL   string `json:"UnsubscribeURL"`
}

// ConfirmSubscriptionResponse contains the XML response of accessing a SubscribeURL
type ConfirmSubscriptionResponse struct {
	XMLName         xml.Name `xml:"ConfirmSubscriptionResponse"`
	SubscriptionArn string   `xml:"ConfirmSubscriptionResult>SubscriptionArn"`
	RequestId       string   `xml:"ResponseMetadata>RequestId"`
}

// UnsubscribeResponse contains the XML response of accessing an UnsubscribeURL
type UnsubscribeResponse struct {
	XMLName   xml.Name `xml:"UnsubscribeResponse"`
	RequestId string   `xml:"ResponseMetadata>RequestId"`
}

// BuildSignature returns a byte array containing a signature usable for SNS verification
func (payload *Payload) BuildSignature() ([]byte, error) {
	var builtSignature bytes.Buffer
	signableKeys := []string{"Message", "MessageId", "Subject", "SubscribeURL", "Timestamp", "Token", "TopicArn", "Type"}
	for _, key := range signableKeys {
		reflectedStruct := reflect.ValueOf(payload)
		field := reflect.Indirect(reflectedStruct).FieldByName(key)
		value := field.String()
		if field.IsValid() && value != "" {
			if key != "Message" && strings.Contains(value, "\n") {
				return nil, fmt.Errorf("SNS field %s contains a newline", key)
			}
			builtSignature.WriteString(key + "\n")
			builtSignature.WriteString(value + "\n")
		}
	}
	return builtSignature.Bytes(), nil
}

// validateSNSURL ensures that a URL provided in an SNS payload points to a
// legitimate AWS SNS endpoint over HTTPS before it is used to make an outbound
// HTTP request. This prevents Server-Side Request Forgery (SSRF) via fields
// such as UnsubscribeURL, which are not covered by the signature in
// BuildSignature() and can therefore be tampered with on an otherwise valid
// AWS-signed message.
func validateSNSURL(rawURL string) error {
	if rawURL == "" {
		return errors.New("URL is empty")
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return err
	}

	if parsedURL.Scheme != "https" {
		return fmt.Errorf("url should be using https")
	}

	if !hostPattern.Match([]byte(parsedURL.Host)) {
		return fmt.Errorf("url is located on an invalid domain")
	}

	return nil
}

// SignatureAlgorithm returns properly Algorithm for AWS Signature Version.
func (payload *Payload) SignatureAlgorithm() x509.SignatureAlgorithm {
	if payload.SignatureVersion == "2" {
		return x509.SHA256WithRSA
	}
	return x509.SHA1WithRSA
}

// Rejecting stale timestamps prevents indefinite replay of captured messages.
// The 1-hour window matches the hard cap on SNS HTTP/S delivery retries ("The
// total policy retry time for an HTTP/S endpoint cannot be greater than 3,600
// seconds"), so no legitimate delivery arrives later than that:
// https://docs.aws.amazon.com/sns/latest/dg/sns-message-delivery-retries.html
func (payload *Payload) validateTimestamp() error {
	t, err := time.Parse(time.RFC3339, payload.Timestamp)
	if err != nil {
		return fmt.Errorf("invalid message timestamp: %w", err)
	}

	if time.Since(t).Abs() > maxTimestampSkew {
		return errors.New("message timestamp is outside the allowed window")
	}

	return nil
}

// VerifyPayload will verify that a payload came from SNS
func (payload *Payload) VerifyPayload() error {
	if err := payload.validateTimestamp(); err != nil {
		return err
	}

	signed, err := payload.BuildSignature()
	if err != nil {
		return err
	}

	payloadSignature, err := base64.StdEncoding.DecodeString(payload.Signature)
	if err != nil {
		return err
	}

	certURL, err := url.Parse(payload.SigningCertURL)
	if err != nil {
		return err
	}

	if certURL.Scheme != "https" {
		return fmt.Errorf("url should be using https")
	}

	if !hostPattern.Match([]byte(certURL.Host)) {
		return fmt.Errorf("certificate is located on an invalid domain")
	}

	// SNS error responses echo query parameters, and pem.Decode returns the first PEM
	// block anywhere in the body, so a certificate passed in the query of an SNS URL
	// would be accepted. Only static .pem files are fetched.
	if certURL.RawQuery != "" || !strings.HasSuffix(certURL.Path, ".pem") {
		return errors.New("certificate URL must be a .pem path without a query")
	}

	resp, err := httpClient.Get(payload.SigningCertURL)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return err
	}

	decodedPem, _ := pem.Decode(body)
	if decodedPem == nil {
		return errors.New("The decoded PEM file was empty!")
	}

	parsedCertificate, err := x509.ParseCertificate(decodedPem.Bytes)
	if err != nil {
		return err
	}

	return parsedCertificate.CheckSignature(payload.SignatureAlgorithm(), signed, payloadSignature)
}

// Subscribe will use the SubscribeURL in a payload to confirm a subscription and return a ConfirmSubscriptionResponse
func (payload *Payload) Subscribe() (ConfirmSubscriptionResponse, error) {
	var response ConfirmSubscriptionResponse
	if payload.SubscribeURL == "" {
		return response, errors.New("Payload does not have a SubscribeURL!")
	}

	if err := validateSNSURL(payload.SubscribeURL); err != nil {
		return response, err
	}

	resp, err := httpClient.Get(payload.SubscribeURL)
	if err != nil {
		return response, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return response, err
	}

	xmlErr := xml.Unmarshal(body, &response)
	if xmlErr != nil {
		return response, xmlErr
	}
	return response, nil
}

// Unsubscribe will use the UnsubscribeURL in a payload to confirm a subscription and return a UnsubscribeResponse
func (payload *Payload) Unsubscribe() (UnsubscribeResponse, error) {
	var response UnsubscribeResponse

	if err := validateSNSURL(payload.UnsubscribeURL); err != nil {
		return response, err
	}

	resp, err := httpClient.Get(payload.UnsubscribeURL)
	if err != nil {
		return response, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return response, err
	}

	xmlErr := xml.Unmarshal(body, &response)
	if xmlErr != nil {
		return response, xmlErr
	}
	return response, nil
}
