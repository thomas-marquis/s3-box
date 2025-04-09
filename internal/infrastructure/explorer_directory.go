package infrastructure

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/thomas-marquis/s3-box/internal/connection"
	"github.com/thomas-marquis/s3-box/internal/explorer"
	"go.uber.org/zap"
)

type S3DirectoryRepositoryImpl struct {
	log     *zap.SugaredLogger
	session *session.Session
	s3      *s3.S3

	conn *connection.Connection
}

var _ explorer.S3DirectoryRepository = &S3DirectoryRepositoryImpl{}

func NewS3DirectoryRepositoryImpl(logger *zap.Logger, defaultConn *connection.Connection) (*S3DirectoryRepositoryImpl, error) {
	log := logger.Sugar()
	var remoteSrc *s3Client
	var err error
	if defaultConn != nil {
		remoteSrc, err = newS3Client(log, defaultConn.AccessKey, defaultConn.SecretKey, defaultConn.Server, defaultConn.BucketName, defaultConn.Region, defaultConn.UseTls)
		if err != nil {
			return nil, fmt.Errorf("NewS3DirectoryRepositoryImpl: %w", err)
		}
	}


    region := 

	if region == "" {
		region = "us-east-1" // for custom endpoints, value is not important but still required
	}
	sess, err := session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(accessKey, secretKey, ""),
		Endpoint:    aws.String(server),
		Logger: aws.LoggerFunc(func(args ...interface{}) {
			log.Infof(args[0].(string), args[1:]...)
		}),
		Region:           aws.String(region),
		S3ForcePathStyle: aws.Bool(true),
		DisableSSL:       aws.Bool(!useTls),
	})
	if err != nil {
		log.Errorf("Error creating session: %v\n", err)
		return nil, fmt.Errorf("newS3Client: %w", err)
	}

	return &S3DirectoryRepositoryImpl{
		log: log,
		s3:  remoteSrc,
	}, nil
}

func (r *S3DirectoryRepositoryImpl) GetByID(ctx context.Context, id explorer.S3DirectoryID) (*explorer.S3Directory, error) {
	if r.s3 == nil {
		return nil, explorer.ErrConnectionNoSet
	}

	var queryPath string
	if id == explorer.RootDirID {
		queryPath = ""
	} else {
		queryPath = strings.TrimPrefix(id.String(), "/") + "/"
	}

	var files = make([]*explorer.S3File, 0)
	var dirs = make([]*explorer.S3Directory, 0)

	if err := r.s3.ListObjectsPagesWithContext(
		ctx,
		&s3.ListObjectsInput{
			Bucket:    aws.String(d.Bucket),
			Prefix:    aws.String(queryPath),
			Delimiter: aws.String("/"),
			MaxKeys:   aws.Int64(10),
		},
		func(page *s3.ListObjectsOutput, lastPage bool) bool {
			for _, obj := range page.Contents {
				key := *obj.Key
				if key == queryPath {
					continue
				}
				newFile := explorer.NewS3File(key)
				newFile.SetSizeBytes(*obj.Size)
				newFile.SetLastModified(*obj.LastModified)
				files = append(files, newFile)
			}

			for _, obj := range page.CommonPrefixes {
				if *obj.Prefix == queryPath {
					continue
				}
				s3Prefix := *obj.Prefix
				isDir := strings.HasSuffix(s3Prefix, "/")
				if isDir {
					dirPathStriped := strings.TrimSuffix(s3Prefix, "/")
					dirPathSplit := strings.Split(dirPathStriped, "/")
					dirName := dirPathSplit[len(dirPathSplit)-1]
					newDir := explorer.NewS3Directory(dirName, currDir)
					dirs = append(dirs, newDir)
				}
			}
			return !lastPage
		},
	); err != nil {
		d.log.Errorf("Error listing objects: %v\n", err)
		return nil, nil, fmt.Errorf("GetDirectoriesAndFileByPath: %w", err)
	}

	// dirs, files, err := r.s3.GetDirectoriesAndFileByPath(ctx, dir)
	// if err != nil {
	// 	r.log.Errorf("error getting directories and files: %v\n", err)
	// 	return nil, nil, fmt.Errorf("ListDirectoryContent: %w", err)
	// }

	return nil, nil
}

func (r *S3DirectoryRepositoryImpl) Save(ctx context.Context, d *explorer.S3Directory) error {
	if r.s3 == nil {
		return explorer.ErrConnectionNoSet
	}
	return nil
}
