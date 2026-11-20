package mongodb_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func mongoDoc(t *testing.T, fields bson.M) bson.D {
	t.Helper()
	raw, err := bson.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	var d bson.D
	if err = bson.Unmarshal(raw, &d); err != nil {
		t.Fatal(err)
	}
	return d
}

func assertStorageError(t *testing.T, err error) {
	t.Helper()
	var commandError mongo.CommandError
	require.ErrorAs(t, err, &commandError)
	require.EqualValues(t, 123, commandError.Code)
}
