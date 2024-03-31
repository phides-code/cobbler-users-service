package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/go-playground/validator"
)

type ResponseStructure struct {
	Data         interface{} `json:"data"`
	ErrorMessage *string     `json:"errorMessage"` // can be string or nil
}

var validate *validator.Validate = validator.New()

var headers = map[string]string{
	"Access-Control-Allow-Origin":  OriginURL,
	"Access-Control-Allow-Headers": "Content-Type",
}

func router(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Printf("Received req %#v", req)

	switch req.HTTPMethod {
	case "GET":
		return processGet(ctx, req)
	case "POST":
		return processPost(ctx, req)
	case "DELETE":
		return processDelete(ctx, req)
	case "PUT":
		return processPut(ctx, req)
	case "OPTIONS":
		return processOptions()
	default:
		return clientError(http.StatusMethodNotAllowed)
	}
}

func processOptions() (events.APIGatewayProxyResponse, error) {
	additionalHeaders := map[string]string{
		"Access-Control-Allow-Methods": "OPTIONS, POST, GET, PUT, DELETE",
		"Access-Control-Max-Age":       "3600",
	}
	mergedHeaders := mergeHeaders(headers, additionalHeaders)

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers:    mergedHeaders,
	}, nil
}

func processGet(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	id, ok := req.PathParameters["id"]
	if !ok {
		return processGetAll(ctx)
	} else {
		return processGetEntityById(ctx, id)
	}
}

func processGetEntityById(ctx context.Context, id string) (events.APIGatewayProxyResponse, error) {
	log.Printf("Received GET entity request with id = %s", id)

	entity, err := getEntity(ctx, id)
	if err != nil {
		return serverError(err)
	}

	if entity == nil {
		return clientError(http.StatusNotFound)
	}

	response := ResponseStructure{
		Data:         entity,
		ErrorMessage: nil,
	}

	responseJson, err := json.Marshal(response)
	if err != nil {
		return serverError(err)
	}
	log.Printf("Successfully fetched entity %s", response.Data)

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(responseJson),
		Headers:    headers,
	}, nil
}

func processGetAll(ctx context.Context) (events.APIGatewayProxyResponse, error) {
	log.Print("Received GET entities request")

	entities, err := listEntities(ctx)
	if err != nil {
		return serverError(err)
	}

	response := ResponseStructure{
		Data:         entities,
		ErrorMessage: nil,
	}

	responseJson, err := json.Marshal(response)
	if err != nil {
		return serverError(err)
	}
	log.Printf("Successfully fetched entities: %s", response.Data)

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(responseJson),
		Headers:    headers,
	}, nil
}

func processPost(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var createdEntity NewOrUpdatedEntity
	err := json.Unmarshal([]byte(req.Body), &createdEntity)
	if err != nil {
		log.Printf("Can't unmarshal body: %v", err)
		return clientError(http.StatusUnprocessableEntity)
	}

	err = validate.Struct(&createdEntity)
	if err != nil {
		log.Printf("Invalid body: %v", err)
		return clientError(http.StatusBadRequest)
	}
	log.Printf("Received POST request with entity: %+v", createdEntity)

	entity, err := insertEntity(ctx, createdEntity)
	if err != nil {
		return serverError(err)
	}
	log.Printf("Inserted new entity: %+v", entity)

	response := ResponseStructure{
		Data:         entity,
		ErrorMessage: nil,
	}

	responseJson, err := json.Marshal(response)
	if err != nil {
		return serverError(err)
	}
	log.Printf("Successfully fetched entity %s", response.Data)

	additionalHeaders := map[string]string{
		"Location": fmt.Sprintf("/%s/%s", ApiPath, entity.Id),
	}
	mergedHeaders := mergeHeaders(headers, additionalHeaders)

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusCreated,
		Body:       string(responseJson),
		Headers:    mergedHeaders,
	}, nil
}

func processDelete(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	id, ok := req.PathParameters["id"]
	if !ok {
		return clientError(http.StatusBadRequest)
	}
	log.Printf("Received DELETE request with id = %s", id)

	entity, err := deleteEntity(ctx, id)
	if err != nil {
		return serverError(err)
	}

	if entity == nil {
		return clientError(http.StatusNotFound)
	}

	response := ResponseStructure{
		Data:         entity,
		ErrorMessage: nil,
	}

	responseJson, err := json.Marshal(response)
	if err != nil {
		return serverError(err)
	}

	log.Printf("Successfully deleted entity %+v", entity)

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(responseJson),
		Headers:    headers,
	}, nil
}

func processPut(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	id, ok := req.PathParameters["id"]
	if !ok {
		return clientError(http.StatusBadRequest)
	}

	var updatedEntity NewOrUpdatedEntity
	err := json.Unmarshal([]byte(req.Body), &updatedEntity)
	if err != nil {
		log.Printf("Can't unmarshal body: %v", err)
		return clientError(http.StatusUnprocessableEntity)
	}

	err = validate.Struct(&updatedEntity)
	if err != nil {
		log.Printf("Invalid body: %v", err)
		return clientError(http.StatusBadRequest)
	}
	log.Printf("Received PUT request with entity: %+v", updatedEntity)

	entity, err := updateEntity(ctx, id, updatedEntity)
	if err != nil {
		return serverError(err)
	}

	if entity == nil {
		return clientError(http.StatusNotFound)
	}

	response := ResponseStructure{
		Data:         entity,
		ErrorMessage: nil,
	}

	responseJson, err := json.Marshal(response)
	if err != nil {
		return serverError(err)
	}

	log.Printf("Updated entity: %+v", entity)

	additionalHeaders := map[string]string{
		"Location": fmt.Sprintf("/%s/%s", ApiPath, entity.Id),
	}
	mergedHeaders := mergeHeaders(headers, additionalHeaders)

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(responseJson),
		Headers:    mergedHeaders,
	}, nil
}
