package s3client

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go/logging"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/notification"
)

type awsClient struct {
	*baseApiImpl

	notifier notification.Repository
	logger   *log.Logger
}

func newAwsClient(conn *connection_deck.Connection, notifier notification.Repository, opts ...func(*s3.Options)) Client {
	logger := log.New(os.Stdout, conn.ID().String(), log.LstdFlags)
	client := s3.New(s3.Options{
		Credentials:  credentials.NewStaticCredentialsProvider(conn.AccessKey(), conn.SecretKey(), ""),
		Region:       conn.Region(),
		Logger:       logging.NewStandardLogger(logger.Writer()),
		UsePathStyle: true,
	}, opts...)

	return newClientImpl(client, conn.Bucket(), &awsClient{
		baseApiImpl: newBaseApiImpl(client, conn.Bucket()),
		notifier:    notifier,
		logger:      logger,
	})
}

func (c *awsClient) GetObjectGrants(ctx context.Context, key string) (Grants, error) {
	aclRes, err := c.client.GetObjectAcl(ctx, &s3.GetObjectAclInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return Grants{}, err
	}

	var (
		grantRead        []string
		grantReadAcp     []string
		grantWriteAcp    []string
		grantFullControl []string
	)

	for _, grant := range aclRes.Grants {
		switch grant.Permission {
		case types.PermissionRead:
			grantRead = append(grantRead, generatePermissionGrant(grant.Grantee))
		case types.PermissionReadAcp:
			grantReadAcp = append(grantReadAcp, generatePermissionGrant(grant.Grantee))
		case types.PermissionWriteAcp:
			grantWriteAcp = append(grantWriteAcp, generatePermissionGrant(grant.Grantee))
		case types.PermissionFullControl:
			grantFullControl = append(grantFullControl, generatePermissionGrant(grant.Grantee))
		case types.PermissionWrite:
			c.notifier.NotifyDebug("ignoring write permission at object level")
		default:
			c.notifier.NotifyDebug(fmt.Sprintf("unknown permission for grant: %s", grant.Permission))
		}
	}

	return Grants{
		Read:        grantRead,
		ReadAcp:     grantReadAcp,
		WriteAcp:    grantWriteAcp,
		FullControl: grantFullControl,
	}, nil

}

func generatePermissionGrant(grantee *types.Grantee) string {
	if grantee.URI != nil {
		return "uri=" + *grantee.URI
	}
	if grantee.ID != nil {
		return "id=" + *grantee.ID
	}
	if grantee.EmailAddress != nil {
		return "emailAddress=" + *grantee.EmailAddress
	}
	return ""
}
