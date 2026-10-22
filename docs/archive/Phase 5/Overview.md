Phase 4 was abandoned midway for a pivot to a cloud backend with lambdas in Go and other AWS deployment styles. The constraint of CPP and GCP delayed progress. While building a web server in cpp was a fun exercise it was ultimately unproductive.

The project V5 is intended to be deployed entirely on AWS using full cloud management with apigatev2 for both the http gateway and the websockets for live chat.

Cognito will be used for user management with a simple setup. No cross account interactions or things like that will be made available.

