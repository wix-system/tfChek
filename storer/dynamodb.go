package storer

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/spf13/viper"
	"strconv"
	"tfChek/misc"
)

const (
	SEQUENCEKEY     = "Sequence"
	SEQUENCENAMEKEY = "Key"
	SEQUENCENAME    = "tfChek-global"
)

func CreateSequenceTable() error {
	tableName := viper.GetString(misc.AWSSequenceTable)
	return createSequenceTable(tableName)
}

func getSession() (*session.Session, error) {
	region := viper.GetString(misc.AWSRegion)
	s, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	if err != nil {
		misc.Debugf("Cannot create DynamoDB session. Error: %s", err)
		return nil, err
	}
	return s, nil
}

func createSequenceTable(name string) error {
	s, err := getSession()
	if err != nil {
		return err
	}
	svc := dynamodb.New(s)
	input := &dynamodb.CreateTableInput{
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String(SEQUENCENAMEKEY),
				AttributeType: aws.String(dynamodb.ScalarAttributeTypeS),
			},
			//{
			//	AttributeName: aws.String(SEQUENCEKEY),
			//	AttributeType: aws.String(dynamodb.ScalarAttributeTypeN),
			//},
		},
		KeySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String(SEQUENCENAMEKEY),
				KeyType:       aws.String(dynamodb.KeyTypeHash),
			},
			//{
			//	AttributeName: aws.String(SEQUENCEKEY),
			//	KeyType:       aws.String(dynamodb.KeyTypeRange),
			//},
		},
		BillingMode: aws.String(dynamodb.BillingModePayPerRequest),
		//ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
		//	ReadCapacityUnits:  aws.Int64(1),
		//	WriteCapacityUnits: aws.Int64(1),
		//},
		TableName: aws.String(name),
	}

	result, err := svc.CreateTable(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case dynamodb.ErrCodeResourceInUseException:
				fmt.Println(dynamodb.ErrCodeResourceInUseException, aerr.Error())
			case dynamodb.ErrCodeLimitExceededException:
				fmt.Println(dynamodb.ErrCodeLimitExceededException, aerr.Error())
			case dynamodb.ErrCodeInternalServerError:
				fmt.Println(dynamodb.ErrCodeInternalServerError, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return err
	}
	fmt.Println(result)
	err = svc.WaitUntilTableExists(&dynamodb.DescribeTableInput{TableName: aws.String(name)})
	if err != nil {
		misc.Debugf("Failed to wait until table exists. Error: %s", err)
		return err
	}
	return nil
}
func DeleteSequenceTable() error {
	tableName := viper.GetString(misc.AWSSequenceTable)
	return deleteSequenceTable(tableName)
}

func deleteSequenceTable(name string) error {
	s, err := getSession()
	if err != nil {
		return err
	}
	svc := dynamodb.New(s)
	input := &dynamodb.DeleteTableInput{
		TableName: aws.String(name),
	}
	result, err := svc.DeleteTable(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case dynamodb.ErrCodeResourceInUseException:
				fmt.Println(dynamodb.ErrCodeResourceInUseException, aerr.Error())
			case dynamodb.ErrCodeResourceNotFoundException:
				fmt.Println(dynamodb.ErrCodeResourceNotFoundException, aerr.Error())
			case dynamodb.ErrCodeLimitExceededException:
				fmt.Println(dynamodb.ErrCodeLimitExceededException, aerr.Error())
			case dynamodb.ErrCodeInternalServerError:
				fmt.Println(dynamodb.ErrCodeInternalServerError, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return err
	}

	fmt.Println(result)
	err = svc.WaitUntilTableNotExists(&dynamodb.DescribeTableInput{TableName: aws.String(name)})
	if err != nil {
		misc.Debugf("Failed to wait until table does not exist. Error: %s", err)
		return err
	}
	return nil
}

func ListSequenceTable() (bool, error) {
	tableName := viper.GetString(misc.AWSSequenceTable)
	return listSequenceTable(tableName)
}

//TODO: use paginator here to list all tables
func listSequenceTable(name string) (bool, error) {
	s, err := getSession()
	if err != nil {
		return false, err
	}
	svc := dynamodb.New(s)
	result, err := svc.ListTables(nil)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case dynamodb.ErrCodeInternalServerError:
				fmt.Println(dynamodb.ErrCodeInternalServerError, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return false, err
	}
	fmt.Println(result)
	var found bool = false
	for _, i := range result.TableNames {
		if *i == name {
			found = true
			break
		}
	}
	return found, nil
}

func EnsureSequenceTable() error {
	tableName := viper.GetString(misc.AWSSequenceTable)
	return checkSequenceTable(tableName)
}

func checkSequenceTable(name string) error {
	exists, err := listSequenceTable(name)
	if err != nil {
		return err
	}
	if !exists {
		err := createSequenceTable(name)
		if err != nil {
			return err
		}
	}
	return nil
}

func updateSequence(seq int, name string) error {
	s, err := getSession()
	if err != nil {
		return err
	}
	svc := dynamodb.New(s)
	input := &dynamodb.PutItemInput{
		Item: map[string]*dynamodb.AttributeValue{
			SEQUENCENAMEKEY: {S: aws.String(SEQUENCENAME)},
			SEQUENCEKEY:     {N: aws.String(strconv.Itoa(seq))},
		},
		ReturnConsumedCapacity: aws.String("TOTAL"),
		TableName:              aws.String(name),
	}

	result, err := svc.PutItem(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case dynamodb.ErrCodeConditionalCheckFailedException:
				fmt.Println(dynamodb.ErrCodeConditionalCheckFailedException, aerr.Error())
			case dynamodb.ErrCodeProvisionedThroughputExceededException:
				fmt.Println(dynamodb.ErrCodeProvisionedThroughputExceededException, aerr.Error())
			case dynamodb.ErrCodeResourceNotFoundException:
				fmt.Println(dynamodb.ErrCodeResourceNotFoundException, aerr.Error())
			case dynamodb.ErrCodeItemCollectionSizeLimitExceededException:
				fmt.Println(dynamodb.ErrCodeItemCollectionSizeLimitExceededException, aerr.Error())
			case dynamodb.ErrCodeTransactionConflictException:
				fmt.Println(dynamodb.ErrCodeTransactionConflictException, aerr.Error())
			case dynamodb.ErrCodeRequestLimitExceeded:
				fmt.Println(dynamodb.ErrCodeRequestLimitExceeded, aerr.Error())
			case dynamodb.ErrCodeInternalServerError:
				fmt.Println(dynamodb.ErrCodeInternalServerError, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return err
	}

	fmt.Println(result)
	return nil
}

func getSequence(name string) (int, error) {
	s, err := getSession()
	if err != nil {
		return -1, err
	}
	svc := dynamodb.New(s)

	input := &dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			SEQUENCENAMEKEY: {S: aws.String(SEQUENCENAME)},
			//SEQUENCEKEY:{N:nil},
		},
		TableName:      aws.String(name),
		ConsistentRead: aws.Bool(true),
	}

	result, err := svc.GetItem(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case dynamodb.ErrCodeProvisionedThroughputExceededException:
				fmt.Println(dynamodb.ErrCodeProvisionedThroughputExceededException, aerr.Error())
			case dynamodb.ErrCodeResourceNotFoundException:
				fmt.Println(dynamodb.ErrCodeResourceNotFoundException, aerr.Error())
			case dynamodb.ErrCodeRequestLimitExceeded:
				fmt.Println(dynamodb.ErrCodeRequestLimitExceeded, aerr.Error())
			case dynamodb.ErrCodeInternalServerError:
				fmt.Println(dynamodb.ErrCodeInternalServerError, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return -1, err
	}
	fmt.Println(result)
	seqItem, ok := result.Item[SEQUENCEKEY]
	if ok {
		v := *seqItem.N
		seq, err := strconv.Atoi(v)
		if err != nil {
			msg := fmt.Sprintf("Cannot convert value %s to integer", v)
			misc.Debug(msg)
			return -1, errors.New(msg)
		}
		return seq, nil
	} else {
		misc.Debugf("Sequence field is absent")
		return -1, errors.New("sequence field is absent")
	}
}
