package s3client

import (
	"context"
	"crypto/tls"
	"errors"
	"log"
	http2 "net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/aws/smithy-go/logging"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
)

type awsClient struct {
	*baseApiImpl

	logger *log.Logger
}

func NewAwsClient(conn *connection_deck.Connection, opts ...func(*s3.Options)) Client {
	logger := log.New(os.Stdout, conn.ID().String(), log.LstdFlags)

	var baseEndpoint *string
	if conn.Server() != "" {
		server := conn.Server()
		if !strings.HasPrefix(server, "http://") && !strings.HasPrefix(server, "https://") {
			protocol := "http://"
			if conn.IsTLSActivated() {
				protocol = "https://"
			}
			server = protocol + server
		}
		baseEndpoint = aws.String(server)
	}

	region := conn.Region()
	if region == "" {
		region = "us-east-1"
	}

	client := s3.New(s3.Options{
		Credentials:  credentials.NewStaticCredentialsProvider(conn.AccessKey(), conn.SecretKey(), ""),
		Region:       region,
		BaseEndpoint: baseEndpoint,
		Logger:       logging.NewStandardLogger(logger.Writer()),
		UsePathStyle: true,
		HTTPClient:   &http2.Client{Transport: &http2.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}},
	}, opts...)

	return newClientImpl(client, conn.Bucket(), &awsClient{
		baseApiImpl: newBaseApiImpl(client, conn.Bucket()),
		logger:      logger,
	})
}

func (c *awsClient) GetObjectGrants(ctx context.Context, key string, opts ...Option) (Grants, error) {
	in := &s3.GetObjectAclInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}
	for _, opt := range opts {
		opt(in)
	}
	aclRes, err := c.client.GetObjectAcl(ctx, in)
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			switch apiErr.ErrorCode() {
			case "AccessDenied", "PermissionDenied":
				return Grants{}, nil
			}
		}

		var respErr *http.ResponseError
		if errors.As(err, &respErr) && respErr.HTTPStatusCode() == 403 {
			return Grants{}, nil
		}

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
