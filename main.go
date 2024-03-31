package main

import (
	"log"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

var db dynamodb.Client

func init() {
	dbClient, err := getClient()

	if err != nil {
		log.Fatal(err)
	}

	db = dbClient
}

func main() {
	lambda.Start(router)
}
