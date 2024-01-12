package main

import (
	"encoding/json"
	"os"

	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/prow/config/secret"
	"k8s.io/test-infra/prow/github"

	"cncf.io/infra/verify-conformance-release/pkg/plugin"
)

func main() {
	log := logrus.StandardLogger().WithField("plugin", "verify-conformance-release")
	githubToken, ok := os.LookupEnv("GITHUB_TOKEN")
	if !ok {
		log.Fatalf("error: unable to find environment variable: GITHUB_TOKEN")
	}
	eventFilePath, ok := os.LookupEnv("GITHUB_EVENT_PATH")
	if !ok {
		log.Fatalf("error: unable to find environment variable: GITHUB_EVENT_PATH")
	}
	tokenFile, err := os.CreateTemp("", "ghtoken-*")
	if err != nil {
		log.WithError(err).Fatalf("error: failed to create new temp file for token")
	}
	defer func() {
		if _, err := os.ReadFile(tokenFile.Name()); err != nil {
			log.WithError(err).Fatalf("error: failed to remove temp github token file")
		}
	}()
	if _, err := tokenFile.Write([]byte(githubToken)); err != nil {
		log.WithError(err).Fatalf("error: failed to write to temp token file")
	}
	if err := secret.Add(tokenFile.Name()); err != nil {
		logrus.WithError(err).Fatal("error: starting test-infra/prow/config/secret agent")
	}
	gen := secret.GetTokenGenerator(tokenFile.Name())
	ghc := github.NewClient(gen, secret.Censor, github.DefaultGraphQLEndpoint, github.DefaultAPIEndpoint)
	if err := ghc.Throttle(360, 360); err != nil {
		logrus.WithError(err).Fatal("error: throttling GitHub client")
	}
	var pullRequestEvent *github.PullRequestEvent
	eventFileBytes, err := os.ReadFile(eventFilePath)
	if err != nil {
		log.WithError(err).Fatalf("error: unable to read event file at path '%v'", eventFilePath)
	}
	if err := json.Unmarshal(eventFileBytes, &pullRequestEvent); err != nil {
		log.WithError(err).Fatalf("error: failed to parse event file into pull request")
	}

	if err := plugin.HandlePullRequestEvent(log, ghc, pullRequestEvent); err != nil {
		log.WithError(err).Fatalf("error: failed to handle pull request event")
	}
}
