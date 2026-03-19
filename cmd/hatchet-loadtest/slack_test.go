package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

func TestSendSlack(t *testing.T) {
	slackSender := NewSlackSender("hatchet-staging-loadtest-us-west-2", os.Getenv("LOAD_TEST_SLACK_WEBHOOK_URL"))
	bytes, err := ioutil.ReadFile("duration_plot.png")
	err = slackSender.SendToSlack(bytes)
	if err != nil {
		fmt.Println(err)
	}
}
