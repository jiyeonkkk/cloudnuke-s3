package aws

import (
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

func getAllS3buckets(session *session.Session, region string, excludeAfter time.Time) ([]*string, error) {
	svc := s3.New(session)
	result, err := svc.ListBuckets(&s3.ListBucketsInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	//https://docs.aws.amazon.com/ko_kr/AmazonS3/latest/API/RESTServiceGET.html
	var names []*string
	for _, buckets := range result.Buckets {
		if excludeAfter.After(*buckets.CreatedTime) {
			names = append(names, buckets.Name)
		}
	}
	//bucket name -> names
	return names, nil
}

func nukeAllS3buckets(session *session.Session, names []*string) error{
	svc := s3.New(session)

	//bucket이 없으면
	if len(names) == 0 {
		logging.Logger.Infof("No S3 buckets to nuke in region %s", *session.Config.Region)
		return nil
	}

	//bucket이 있으면
	logging.Logger.Infof("Deleting all S3 buckets in region %s", *session.Config.Region)
	var deletedNames []*string

	//bucket삭제
	for _, name := range names {
		params := &s3.DeleteBucketInput{
			Name: name,
		}
		
		//params==*DeleteBucketInput
		_, err := svc.DeleteBucket(params)
		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
		} else {
			deletedNames = append(deletedNames, name)
			logging.Logger.Infof("Deleted S3 Bucket: %s", *name)
		}
	}

	//제거될 Bucket이 있으면
	if len(deletedNames) > 0 {

	}

	logging.Logger.Infof("[OK] %d S3 Bucket(s) deleted in %s", len(deletedNames), *session.Config.Region)
	return nil

}


