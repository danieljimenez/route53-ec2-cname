package main

import (
	"fmt"
	"github.com/akamensky/argparse"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/route53"
	"os"
	"strings"
)

var sess *session.Session

func findInstance(instanceName *string) (*string, error) {
	ec2Client := ec2.New(sess)

	fmt.Println(fmt.Sprintf("⏳ looking for ec2 instance tagged '%s'...", *instanceName))
	instancesOutput, err := ec2Client.DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("tag:Name"),
				Values: []*string{instanceName},
			},
			{
				Name: aws.String("instance-state-name"),
				Values: []*string{
					aws.String("running"),
				},
			},
		},
	})

	// Notice how else's can be dropped

	if len(instancesOutput.Reservations) == 0 {
		return nil, fmt.Errorf("⛔ no running instances found by tag '%s' in this region️", instanceName)
	}
	if len(instancesOutput.Reservations) > 1 {
		return nil, fmt.Errorf("⛔ more than one running instance found by tag '%s' in this region", instanceName)
	}
	fmt.Println(fmt.Sprintf("✅ found ec2 instance %s...", *instancesOutput.Reservations[0].Instances[0].InstanceId))
	return instancesOutput.Reservations[0].Instances[0].PublicDnsName, err
}

func findHostedZoneId(domain *string) (*string, error) {
	domainString := *domain

	// If the last character is not a ".", append one.
	dsLastChar := domainString[len(domainString)-1:]
	if dsLastChar != "." {
		domainString += "."
	}

	// List all the zones, with the closest matching one in the 0 position.
	route53Client := route53.New(sess)

	fmt.Println(fmt.Sprintf("⏳ looking for hosted zone '%s'...", domainString))
	output, err := route53Client.ListHostedZonesByName(&route53.ListHostedZonesByNameInput{
		DNSName: &domainString,
	})
	if err != nil {
		return nil, err
	}

	if len(output.HostedZones) == 0 {
		return nil, fmt.Errorf("⛔ unable to find hosted zone: '%s'", domainString)
	}
	idPath := output.HostedZones[0].Id

	// Strip the extraneous name space in the ID
	id := strings.Split(*idPath, "/hostedzone/")
	hostedZoneId := id[1]

	fmt.Println(fmt.Sprintf("✅ found hosted zone %s", hostedZoneId))
	return &hostedZoneId, nil
}

func changeRecordSet(hostedZoneId *string, targetRecord *string, dnsName *string) error {
	route53Client := route53.New(sess)

	fmt.Println("⏳ submitting change...")
	changeResourceRecordSetOutput, err := route53Client.ChangeResourceRecordSets(&route53.ChangeResourceRecordSetsInput{
		HostedZoneId: hostedZoneId,
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{
				{
					Action: aws.String("UPSERT"),
					ResourceRecordSet: &route53.ResourceRecordSet{
						Name: targetRecord,
						ResourceRecords: []*route53.ResourceRecord{
							{
								Value: dnsName,
							},
						},
						TTL:  aws.Int64(5),
						Type: aws.String("CNAME"),
					},
				},
			},
		},
	})
	if err != nil {
		return err
	}

	fmt.Println("⏳ change submitted, awaiting propagation...")
	return route53Client.WaitUntilResourceRecordSetsChanged(&route53.GetChangeInput{
		Id: changeResourceRecordSetOutput.ChangeInfo.Id,
	})
}

func main() {
	// Parse arguments using argparse.
	parser := argparse.NewParser(os.Args[0], "manages route53 records by updating them with an ec2 instance's public cname")

	instanceName := parser.String("n", "name", &argparse.Options{
		Required: true,
		Help:     "the name of the instance you'd like to use, this tool will grab it's public dns name",
	})

	targetRecord := parser.String("r", "record", &argparse.Options{
		Required: true,
		Help:     "the route53 resource you'd like to update to point to an ec2 instance",
	})

	// Using a go-idiomic one-liner conditional construct
	if err := parser.Parse(os.Args); err != nil {
		// Print usage if arguments are missing.
		fmt.Print(parser.Usage(err))
		os.Exit(1)
	}

	// Create an AWS Session using the user or system's shared config.
	sess = session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	domainName := strings.SplitAfterN(*targetRecord, ".", 2)
	hostedZoneId, err := findHostedZoneId(&domainName[1])
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	dnsName, err := findInstance(instanceName)

	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	if err := changeRecordSet(hostedZoneId, targetRecord, dnsName); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	fmt.Println("🆒 ㊗️ mission successful")
	os.Exit(0)
}
