package dynamodb

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/dynamodbiface"
	"github.com/pkg/errors"

	commonpb "github.com/kinecosystem/kin-api/genproto/common/v3"

	"github.com/kinecosystem/agora-transaction-services-internal/pkg/invoice"
)

type db struct {
	db dynamodbiface.ClientAPI
}

// New returns a dynamo-backed invoice.Store
func New(client dynamodbiface.ClientAPI) invoice.Store {
	return &db{
		db: client,
	}
}

// Add implements invoice.Store.Add.
func (d *db) Add(ctx context.Context, inv *commonpb.Invoice, txHash []byte) error {
	item, err := toItem(inv, txHash)
	if err != nil {
		return err
	}

	_, err = d.db.PutItemRequest(&dynamodb.PutItemInput{
		TableName:           tableNameStr,
		Item:                item,
		ConditionExpression: putConditionStr,
	}).Send(ctx)
	if err != nil {
		if aErr, ok := err.(awserr.Error); ok {
			switch aErr.Code() {
			case dynamodb.ErrCodeConditionalCheckFailedException:
				return invoice.ErrExists
			}
		}

		return errors.Wrapf(err, "failed to store invoice")
	}

	return nil
}

// Get implements invoice.Store.Get.
func (d *db) Get(ctx context.Context, invoiceHash []byte, txHash []byte) (*commonpb.Invoice, error) {
	if len(invoiceHash) != 28 {
		return nil, errors.Errorf("invalid invoice hash len: %d", len(invoiceHash))
	}

	if len(txHash) != 32 {
		return nil, errors.Errorf("invalid transaction hash len: %d", len(txHash))
	}

	resp, err := d.db.GetItemRequest(&dynamodb.GetItemInput{
		TableName: tableNameStr,
		Key: map[string]dynamodb.AttributeValue{
			tableHashKey: {
				B: invoiceHash,
			},
			tableRangeKey: {
				B: txHash,
			},
		},
	}).Send(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get invoice")
	}

	if len(resp.Item) == 0 {
		return nil, invoice.ErrNotFound
	}
	return fromItem(resp.Item)
}

// Exists implements invoice.Store.Exists
func (d *db) Exists(ctx context.Context, invoiceHash []byte) (bool, error) {
	input := &dynamodb.QueryInput{
		TableName:              tableNameStr,
		KeyConditionExpression: existsKeyConditionStr,
		ExpressionAttributeValues: map[string]dynamodb.AttributeValue{
			":invoice_hash": {B: invoiceHash},
		},
		Limit: aws.Int64(1), // Given the put condition, only 1 should exist
	}

	resp, err := d.db.QueryRequest(input).Send(ctx)
	if err != nil {
		return false, errors.Wrap(err, "failed to query invoices")
	}

	return len(resp.Items) > 0, nil
}
