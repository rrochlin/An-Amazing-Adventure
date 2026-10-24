// Command ws-chat is the ws-chat Lambda; the handler is in internal/lambdas/wschat.
package main

import (
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/rrochlin/an-amazing-adventure/internal/lambdas/wschat"
)

func main() {
	lambda.Start(wschat.Handler)
}
