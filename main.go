package main

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/iam"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// https://www.pulumi.com/blog/managing-aws-credentials-on-cicd-part-1/

// Action:
// Allow anybody (i.e., members of the group) to call the sts:AssumeRole API.
// This allows them to "assume the role" of a more permissive IAM Role
// when they go to update a stack later.

// Resource:
// This is the set of resources that the "sts:AssumeRole" operation could be
// performed on, which is to say any IAM role in the current AWS account.
func SetPolicy(account string) string {
	return fmt.Sprintf(`{
				"Version": "2012-10-17",
				"Statement": [
				  {
					"Action": [
					  "sts:AssumeRole"
					],
					"Effect": "Allow",
					"Resource": "arn:aws:iam::%s:role/*",
					"Sid": ""
				  }
				]
			  }`, account)
}

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {

		// function to access to the effective Account ID, User ID, and ARN in which this provider is authorized
		current, err := aws.GetCallerIdentity(ctx, nil, nil)
		if err != nil {
			return err
		}

		// Create CiCD user
		lbUser, err := iam.NewUser(ctx, "lbUser", &iam.UserArgs{
			Name: pulumi.String("Jenkins-pulumi-bot"),
			Path: pulumi.String("/system/"),
			Tags: pulumi.StringMap{
				"purpose": pulumi.String("Account used to perform Pulumi stack updates on CI/CD."),
			},
		})
		if err != nil {
			return err
		}

		// Provides an IAM access key. This is a set of credentials that allow API requests to be made as an IAM user.
		_, err = iam.NewAccessKey(ctx, "lbAccessKey", &iam.AccessKeyArgs{
			User: lbUser.Name,
		})
		if err != nil {
			return err
		}

		// an IAM group.
		grp, err := iam.NewGroup(ctx, "pulumiStackUpdaters", &iam.GroupArgs{
			Name: pulumi.String("pulumiStackUpdaters"),
		})
		if err != nil {
			return err
		}

		// Provides a top level resource to manage IAM Group membership for IAM Users.
		_, err = iam.NewGroupMembership(ctx, "cicdUserMembership", &iam.GroupMembershipArgs{
			Group: grp.Name,
			Name:  pulumi.String("cicdUserMembership"),
			Users: pulumi.StringArray{
				lbUser.Name,
			},
		})

		// create policy. Pass in account number
		pol := SetPolicy(current.AccountId)
		_, err = iam.NewGroupPolicy(ctx, "pulumiStackUpdatersPolicy", &iam.GroupPolicyArgs{
			Group:  grp.Name,
			Name:   pulumi.String("pulumiStackUpdatersPolicy"),
			Policy: pulumi.String(pol),
		})

		return nil
	})
}
