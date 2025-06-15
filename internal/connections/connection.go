package connections

//
// import (
// 	"strings"
//
// 	"github.com/google/uuid"
// )
//
// func NewEmptyConnection(options ...ConnectionOption) *Connection {
// 	return New("", "", "", "", options...)
// }
//
// func (c *Connection) Update(options ...ConnectionOption) {
// 	for _, opt := range options {
// 		opt(c)
// 	}
// 	c.IncRevision()
// }
//
// // Compare checks if the current connection is equal to another connection.
// func (c *Connection) Compare(other *Connection) bool {
// 	return c.Name == other.Name &&
// 		c.Server == other.Server &&
// 		c.SecretKey == other.SecretKey &&
// 		c.AccessKey == other.AccessKey &&
// 		c.BucketName == other.BucketName &&
// 		c.UseTls == other.UseTls &&
// 		c.Region == other.Region &&
// 		c.Type == other.Type &&
// 		c.ReadOnly == other.ReadOnly &&
// 		c.revision == other.revision
// }
//
// type ConnectionExport struct {
// 	JSONData []byte
// 	Count    int
// }
