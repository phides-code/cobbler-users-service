package main

import (
	"context"
	"errors"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/smithy-go"
	"github.com/google/uuid"
)

type Entity struct {
	Id              string   `json:"id" dynamodbav:"id"`
	Nickname        string   `json:"nickname" dynamodbav:"nickname"`
	Username        string   `json:"username" dynamodbav:"username"`
	Picture         string   `json:"picture" dynamodbav:"picture"`
	Updated_at      string   `json:"updated_at" dynamodbav:"updated_at"`
	Email           string   `json:"email" dynamodbav:"email"`
	Email_verified  bool     `json:"email_verified" dynamodbav:"email_verified"`
	Sub             string   `json:"sub" dynamodbav:"sub"`
	AuthoredRecipes []string `json:"authoredRecipes" dynamodbav:"authoredRecipes"`
	LikedRecipes    []string `json:"likedRecipes" dynamodbav:"likedRecipes"`
}

type NewOrUpdatedEntity struct {
	Nickname        string   `json:"nickname" validate:"required"`
	Username        string   `json:"username" validate:"required"`
	Picture         string   `json:"picture" validate:"required"`
	Updated_at      string   `json:"updated_at" validate:"required"`
	Email           string   `json:"email" validate:"required"`
	Email_verified  bool     `json:"email_verified" validate:"required"`
	Sub             string   `json:"sub" validate:"required"`
	AuthoredRecipes []string `json:"authoredRecipes" validate:"required"`
	LikedRecipes    []string `json:"likedRecipes" validate:"required"`
}

func getClient() (dynamodb.Client, error) {
	sdkConfig, err := config.LoadDefaultConfig(context.TODO())

	dbClient := *dynamodb.NewFromConfig(sdkConfig)

	return dbClient, err
}

func getEntity(ctx context.Context, id string) (*Entity, error) {
	key, err := attributevalue.Marshal(id)
	if err != nil {
		return nil, err
	}

	input := &dynamodb.GetItemInput{
		TableName: aws.String(TableName),
		Key: map[string]types.AttributeValue{
			"id": key,
		},
	}

	log.Printf("Calling DynamoDB with input: %v", input)
	result, err := db.GetItem(ctx, input)
	if err != nil {
		return nil, err
	}
	log.Printf("Executed GetEntity DynamoDb successfully. Result: %#v", result)

	if result.Item == nil {
		return nil, nil
	}

	entity := new(Entity)
	err = attributevalue.UnmarshalMap(result.Item, entity)
	if err != nil {
		return nil, err
	}

	return entity, nil
}

func listEntities(ctx context.Context) ([]Entity, error) {
	entities := make([]Entity, 0)

	var token map[string]types.AttributeValue

	for {
		input := &dynamodb.ScanInput{
			TableName:         aws.String(TableName),
			ExclusiveStartKey: token,
		}

		result, err := db.Scan(ctx, input)
		if err != nil {
			return nil, err
		}

		var fetchedEntity []Entity
		err = attributevalue.UnmarshalListOfMaps(result.Items, &fetchedEntity)
		if err != nil {
			return nil, err
		}

		entities = append(entities, fetchedEntity...)
		token = result.LastEvaluatedKey
		if token == nil {
			break
		}

	}

	return entities, nil
}

func insertEntity(ctx context.Context, newEntity NewOrUpdatedEntity) (*Entity, error) {
	entity := Entity{
		Id:              uuid.NewString(),
		Nickname:        newEntity.Nickname,
		Username:        newEntity.Username,
		Picture:         newEntity.Picture,
		Updated_at:      newEntity.Updated_at,
		Email:           newEntity.Email,
		Email_verified:  newEntity.Email_verified,
		Sub:             newEntity.Sub,
		AuthoredRecipes: newEntity.AuthoredRecipes,
		LikedRecipes:    newEntity.LikedRecipes,
	}

	entityMap, err := attributevalue.MarshalMap(entity)
	if err != nil {
		return nil, err
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(TableName),
		Item:      entityMap,
	}

	res, err := db.PutItem(ctx, input)
	if err != nil {
		return nil, err
	}

	err = attributevalue.UnmarshalMap(res.Attributes, &entity)
	if err != nil {
		return nil, err
	}

	return &entity, nil
}

func deleteEntity(ctx context.Context, id string) (*Entity, error) {
	key, err := attributevalue.Marshal(id)
	if err != nil {
		return nil, err
	}

	input := &dynamodb.DeleteItemInput{
		TableName: aws.String(TableName),
		Key: map[string]types.AttributeValue{
			"id": key,
		},
		ReturnValues: types.ReturnValue(*aws.String("ALL_OLD")),
	}

	res, err := db.DeleteItem(ctx, input)
	if err != nil {
		return nil, err
	}

	if res.Attributes == nil {
		return nil, nil
	}

	entity := new(Entity)
	err = attributevalue.UnmarshalMap(res.Attributes, entity)
	if err != nil {
		return nil, err
	}

	return entity, nil
}

func updateEntity(ctx context.Context, id string, updatedEntity NewOrUpdatedEntity) (*Entity, error) {
	key, err := attributevalue.Marshal(id)
	if err != nil {
		return nil, err
	}

	expr, err := expression.NewBuilder().WithUpdate(
		expression.Set(
			expression.Name("nickname"),
			expression.Value(updatedEntity.Nickname),
		).Set(
			expression.Name("username"),
			expression.Value(updatedEntity.Username),
		).Set(
			expression.Name("picture"),
			expression.Value(updatedEntity.Picture),
		).Set(
			expression.Name("updated_at"),
			expression.Value(updatedEntity.Updated_at),
		).Set(
			expression.Name("email"),
			expression.Value(updatedEntity.Email),
		).Set(
			expression.Name("email_verified"),
			expression.Value(updatedEntity.Email_verified),
		).Set(
			expression.Name("sub"),
			expression.Value(updatedEntity.Sub),
		).Set(
			expression.Name("authoredRecipes"),
			expression.Value(updatedEntity.AuthoredRecipes),
		).Set(
			expression.Name("likedRecipes"),
			expression.Value(updatedEntity.LikedRecipes),
		),
	).WithCondition(
		expression.Equal(
			expression.Name("id"),
			expression.Value(id),
		),
	).Build()
	if err != nil {
		return nil, err
	}

	input := &dynamodb.UpdateItemInput{
		Key: map[string]types.AttributeValue{
			"id": key,
		},
		TableName:                 aws.String(TableName),
		UpdateExpression:          expr.Update(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),

		ConditionExpression: expr.Condition(),
		ReturnValues:        types.ReturnValue(*aws.String("ALL_NEW")),
	}

	res, err := db.UpdateItem(ctx, input)
	if err != nil {
		var smErr *smithy.OperationError
		if errors.As(err, &smErr) {
			var condCheckFailed *types.ConditionalCheckFailedException
			if errors.As(err, &condCheckFailed) {
				return nil, nil
			}
		}

		return nil, err
	}

	if res.Attributes == nil {
		return nil, nil
	}

	entity := new(Entity)
	err = attributevalue.UnmarshalMap(res.Attributes, entity)
	if err != nil {
		return nil, err
	}

	return entity, nil
}
