# About

This tool: 

1. finds an EC2 instance by it's name tag using the `--name` argument.
2. looks up the Route53 hosted zone using the `--record` argument.
3. creates or updates the Route53 resource (using the `--record` argument) with a CNAME pointing to 
the aforementioned EC2 instance address.

# Example

    ./route53-ec2-cname --name my-awesome-server-name --record test.example.com
    
# Notes

* If more than one instance is found, this tool error out.
* The EC2 instance and the Route53 hosted zone must be in the same AWS account at this time.
* This tool uses an AWS "waiter" to wait for the change to propagate throughout Route53, this may give the perception
of the tool freezing. 