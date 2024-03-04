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
)

// https://github.com/robbiet480/go.sns/issues/2
var hostPattern = regexp.MustCompile(`^sns\.[a-zA-Z0-9\-]{3,}\.amazonaws\.com(\.cn)?$`)

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
func (payload *Payload) BuildSignature() []byte {
	var builtSignature bytes.Buffer
	signableKeys := []string{"Message", "MessageId", "Subject", "SubscribeURL", "Timestamp", "Token", "TopicArn", "Type"}
	for _, key := range signableKeys {
		reflectedStruct := reflect.ValueOf(payload)
		field := reflect.Indirect(reflectedStruct).FieldByName(key)
		value := field.String()
		if field.IsValid() && value != "" {
			builtSignature.WriteString(key + "\n")
			builtSignature.WriteString(value + "\n")
		}
	}
	return builtSignature.Bytes()
}

// SignatureAlgorithm returns properly Algorithm for AWS Signature Version.
func (payload *Payload) SignatureAlgorithm() x509.SignatureAlgorithm {
	if payload.SignatureVersion == "2" {
		return x509.SHA256WithRSA
	}
	return x509.SHA1WithRSA
}

// VerifyPayload will verify that a payload came from SNS
func (payload *Payload) VerifyPayload() error {
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

	resp, err := http.Get(payload.SigningCertURL)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
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

	return parsedCertificate.CheckSignature(payload.SignatureAlgorithm(), payload.BuildSignature(), payloadSignature)
}

// Subscribe will use the SubscribeURL in a payload to confirm a subscription and return a ConfirmSubscriptionResponse
func (payload *Payload) Subscribe() (ConfirmSubscriptionResponse, error) {
	var response ConfirmSubscriptionResponse
	if payload.SubscribeURL == "" {
		return response, errors.New("Payload does not have a SubscribeURL!")
	}

	resp, err := http.Get(payload.SubscribeURL)
	if err != nil {
		return response, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
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
	resp, err := http.Get(payload.UnsubscribeURL)
	if err != nil {
		return response, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return response, err
	}

	xmlErr := xml.Unmarshal(body, &response)
	if xmlErr != nil {
		return response, xmlErr
	}
	return response, nil
}
