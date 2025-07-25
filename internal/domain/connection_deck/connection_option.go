package connection_deck

type ConnectionOption func(*Connection)

func AsAWS(region string) ConnectionOption {
	return func(c *Connection) {
		if region == "" || c.provider == AWSProvider {
			return
		}
		c.provider = AWSProvider
		c.region = region
		c.useTLS = true
		c.server = ""
	}
}

func AsS3Like(server string, useTLS bool) ConnectionOption {
	return func(c *Connection) {
		if server == "" || c.provider == S3LikeProvider {
			return
		}
		c.provider = S3LikeProvider
		c.server = server
		c.useTLS = useTLS
		c.region = ""
	}
}

func WithReadOnlyOption(readOnly bool) ConnectionOption {
	return func(c *Connection) {
		c.readOnly = readOnly
	}
}

func WithRevision(revision int) ConnectionOption {
	return func(c *Connection) {
		c.revision = revision
	}
}

func WithUseTLS(useTLS bool) ConnectionOption {
	return func(c *Connection) {
		c.useTLS = useTLS
	}
}

func WithID(id ConnectionID) ConnectionOption {
	return func(c *Connection) {
		c.id = id
	}
}

func WithCredentials(
	accessKey, secretKey string,
) ConnectionOption {
	return func(c *Connection) {
		c.accessKey = accessKey
		c.secretKey = secretKey
	}
}

func WithName(name string) ConnectionOption {
	return func(c *Connection) {
		c.name = name
	}
}

func WithBucket(bucket string) ConnectionOption {
	return func(c *Connection) {
		c.bucket = bucket
	}
}
