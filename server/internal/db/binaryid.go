package db

import (
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// BinaryID is a string that marshals to DynamoDB Binary (B) type.
// The sessions and connections tables were created with B-typed key attributes,
// so key fields must marshal as B rather than the default S.
type BinaryID string

func (b BinaryID) MarshalDynamoDBAttributeValue() (types.AttributeValue, error) {
	return &types.AttributeValueMemberB{Value: []byte(b)}, nil
}

func (b *BinaryID) UnmarshalDynamoDBAttributeValue(av types.AttributeValue) error {
	switch v := av.(type) {
	case *types.AttributeValueMemberB:
		*b = BinaryID(v.Value)
	case *types.AttributeValueMemberS:
		*b = BinaryID(v.Value)
	}
	return nil
}

// binaryIDVal marshals a plain string as a DynamoDB Binary attribute value.
// Used when building expression attribute values for key conditions.
func binaryIDVal(s string) types.AttributeValue {
	return &types.AttributeValueMemberB{Value: []byte(s)}
}

// marshalBinaryKey returns a DynamoDB key map with a single Binary-typed attribute.
// Used to build GetItem/UpdateItem/DeleteItem key inputs for tables whose key
// attribute is stored as B (Binary) type.
func marshalBinaryKey(attr, value string) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		attr: &types.AttributeValueMemberB{Value: []byte(value)},
	}
}

// Ensure BinaryID satisfies the interfaces at compile time.
var _ attributevalue.Marshaler = BinaryID("")
var _ attributevalue.Unmarshaler = (*BinaryID)(nil)
