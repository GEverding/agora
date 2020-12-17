package dynamodb

import (
	"context"
	"crypto/ed25519"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/dynamodbiface"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	accountpb "github.com/kinecosystem/agora-api/genproto/account/v4"

	"github.com/kinecosystem/agora/pkg/account/solana/accountinfo"
)

type cache struct {
	log     *logrus.Entry
	client  dynamodbiface.ClientAPI
	itemTTL time.Duration
}

// New returns a dynamodb-backed accountinfo.Cache
func New(client dynamodbiface.ClientAPI, ttl time.Duration) accountinfo.Cache {
	return &cache{
		log:     logrus.StandardLogger().WithField("type", "app/dynamodb"),
		client:  client,
		itemTTL: ttl,
	}
}

// Get implements accountinfo.Cache.Add
func (c *cache) Put(ctx context.Context, info *accountpb.AccountInfo) error {
	item, err := toItem(info, time.Now().Add(c.itemTTL))
	if err != nil {
		return err
	}

	_, err = c.client.PutItemRequest(&dynamodb.PutItemInput{
		TableName: tableNameStr,
		Item:      item,
	}).Send(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to store account info")
	}

	return nil
}

// Get implements accountinfo.Cache.Get
func (c *cache) Get(ctx context.Context, key ed25519.PublicKey) (*accountpb.AccountInfo, error) {
	resp, err := c.client.GetItemRequest(&dynamodb.GetItemInput{
		TableName: tableNameStr,
		Key: map[string]dynamodb.AttributeValue{
			tableHashKey: {
				B: key,
			},
		},
	}).Send(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get account info")
	}

	if len(resp.Item) == 0 {
		return nil, accountinfo.ErrAccountInfoNotFound
	}

	info, expiry, err := fromItem(resp.Item)
	if err != nil {
		return nil, err
	}

	if expiry.Before(time.Now()) {
		return nil, accountinfo.ErrAccountInfoNotFound
	}

	return info, nil
}