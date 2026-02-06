package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
)

type Todo struct {
	ID        string `json:"id" dynamodbav:"id"`
	Title     string `json:"title" dynamodbav:"title"`
	Content   string `json:"content" dynamodbav:"content"`
	Completed bool   `json:"completed" dynamodbav:"completed"`
	CreatedAt string `json:"createdAt" dynamodbav:"createdAt"`
	UpdatedAt string `json:"updatedAt" dynamodbav:"updatedAt"`
}

var (
	db        *dynamodb.Client
	tableName string
)

func init() {
	tableName = os.Getenv("TABLE_NAME")
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		panic("unable to load SDK config: " + err.Error())
	}
	db = dynamodb.NewFromConfig(cfg)
}

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{
		"Content-Type":                "application/json",
		"Access-Control-Allow-Origin": "*",
	}

	// ルーティング: Resource + HTTPMethod
	switch {
	case req.Resource == "/todos" && req.HTTPMethod == "GET":
		return listTodos(ctx, headers)
	case req.Resource == "/todos/{id}" && req.HTTPMethod == "GET":
		return getTodo(ctx, req.PathParameters["id"], headers)
	case req.Resource == "/todos" && req.HTTPMethod == "POST":
		return createTodo(ctx, req.Body, headers)
	case req.Resource == "/todos/{id}" && req.HTTPMethod == "PUT":
		return updateTodo(ctx, req.PathParameters["id"], req.Body, headers)
	case req.Resource == "/todos/{id}" && req.HTTPMethod == "DELETE":
		return deleteTodo(ctx, req.PathParameters["id"], headers)
	default:
		return respond(http.StatusNotFound, map[string]string{"error": "not found"}, headers)
	}
}

// listTodos は全 TODO を取得する
func listTodos(ctx context.Context, headers map[string]string) (events.APIGatewayProxyResponse, error) {
	out, err := db.Scan(ctx, &dynamodb.ScanInput{
		TableName: aws.String(tableName),
	})
	if err != nil {
		return respond(http.StatusInternalServerError, map[string]string{"error": err.Error()}, headers)
	}

	var todos []Todo
	if err := attributevalue.UnmarshalListOfMaps(out.Items, &todos); err != nil {
		return respond(http.StatusInternalServerError, map[string]string{"error": err.Error()}, headers)
	}

	return respond(http.StatusOK, todos, headers)
}

// getTodo は指定 ID の TODO を取得する
func getTodo(ctx context.Context, id string, headers map[string]string) (events.APIGatewayProxyResponse, error) {
	out, err := db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return respond(http.StatusInternalServerError, map[string]string{"error": err.Error()}, headers)
	}
	if out.Item == nil {
		return respond(http.StatusNotFound, map[string]string{"error": "todo not found"}, headers)
	}

	var todo Todo
	if err := attributevalue.UnmarshalMap(out.Item, &todo); err != nil {
		return respond(http.StatusInternalServerError, map[string]string{"error": err.Error()}, headers)
	}

	return respond(http.StatusOK, todo, headers)
}

// createTodo は新しい TODO を作成する
func createTodo(ctx context.Context, body string, headers map[string]string) (events.APIGatewayProxyResponse, error) {
	var input struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(body), &input); err != nil {
		return respond(http.StatusBadRequest, map[string]string{"error": "invalid request body"}, headers)
	}
	if strings.TrimSpace(input.Title) == "" {
		return respond(http.StatusBadRequest, map[string]string{"error": "title is required"}, headers)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	todo := Todo{
		ID:        uuid.New().String(),
		Title:     input.Title,
		Content:   input.Content,
		Completed: false,
		CreatedAt: now,
		UpdatedAt: now,
	}

	item, err := attributevalue.MarshalMap(todo)
	if err != nil {
		return respond(http.StatusInternalServerError, map[string]string{"error": err.Error()}, headers)
	}

	_, err = db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item:      item,
	})
	if err != nil {
		return respond(http.StatusInternalServerError, map[string]string{"error": err.Error()}, headers)
	}

	return respond(http.StatusCreated, todo, headers)
}

// updateTodo は指定 ID の TODO を更新する
func updateTodo(ctx context.Context, id string, body string, headers map[string]string) (events.APIGatewayProxyResponse, error) {
	var input struct {
		Title     *string `json:"title"`
		Content   *string `json:"content"`
		Completed *bool   `json:"completed"`
	}
	if err := json.Unmarshal([]byte(body), &input); err != nil {
		return respond(http.StatusBadRequest, map[string]string{"error": "invalid request body"}, headers)
	}

	now := time.Now().UTC().Format(time.RFC3339)

	// 更新式を動的に構築
	updateParts := []string{"#updatedAt = :updatedAt"}
	exprNames := map[string]string{"#updatedAt": "updatedAt"}
	exprValues := map[string]types.AttributeValue{
		":updatedAt": &types.AttributeValueMemberS{Value: now},
	}

	if input.Title != nil {
		updateParts = append(updateParts, "#title = :title")
		exprNames["#title"] = "title"
		exprValues[":title"] = &types.AttributeValueMemberS{Value: *input.Title}
	}
	if input.Content != nil {
		updateParts = append(updateParts, "#content = :content")
		exprNames["#content"] = "content"
		exprValues[":content"] = &types.AttributeValueMemberS{Value: *input.Content}
	}
	if input.Completed != nil {
		updateParts = append(updateParts, "#completed = :completed")
		exprNames["#completed"] = "completed"
		exprValues[":completed"] = &types.AttributeValueMemberBOOL{Value: *input.Completed}
	}

	updateExpr := "SET " + strings.Join(updateParts, ", ")

	out, err := db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
		UpdateExpression:          aws.String(updateExpr),
		ExpressionAttributeNames:  exprNames,
		ExpressionAttributeValues: exprValues,
		ReturnValues:              types.ReturnValueAllNew,
	})
	if err != nil {
		return respond(http.StatusInternalServerError, map[string]string{"error": err.Error()}, headers)
	}

	var todo Todo
	if err := attributevalue.UnmarshalMap(out.Attributes, &todo); err != nil {
		return respond(http.StatusInternalServerError, map[string]string{"error": err.Error()}, headers)
	}

	return respond(http.StatusOK, todo, headers)
}

// deleteTodo は指定 ID の TODO を削除する
func deleteTodo(ctx context.Context, id string, headers map[string]string) (events.APIGatewayProxyResponse, error) {
	_, err := db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return respond(http.StatusInternalServerError, map[string]string{"error": err.Error()}, headers)
	}

	return respond(http.StatusOK, map[string]string{"message": "deleted"}, headers)
}

// respond は JSON レスポンスを構築するヘルパー
func respond(statusCode int, body interface{}, headers map[string]string) (events.APIGatewayProxyResponse, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers:    headers,
			Body:       `{"error":"internal server error"}`,
		}, nil
	}
	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Headers:    headers,
		Body:       string(jsonBody),
	}, nil
}
