// Command ws-game-action is the ws-game-action Lambda; the handler is in internal/lambdas/wsgameaction.
package main

import (
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/rrochlin/an-amazing-adventure/internal/lambdas/wsgameaction"
)

func main() {
	lambda.Start(wsgameaction.Handler)
}
