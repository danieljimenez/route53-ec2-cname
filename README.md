# About

This tool creates a Route53 resource record, in an existing hosted zone, from an EC2 instance's public DNS address, using it's "name" tag.

# Example

    ./route53-ec2-cname --name my-awesome-server-name --record test.example.com