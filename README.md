# AWS events exporter

This application exports information about AWS scheduled events.
It only supports currently IAM roles, and doesnt need any other permission.

IAM permission required:
`ec2:DescribeInstanceStatus`

Travis Build

[![Build Status](https://travis-ci.org/Kronin-Cloud/aws-events-exporter.svg?branch=master)](https://travis-ci.org/Kronin-Cloud/aws-events-exporter)

The exporter does following:
Exports metrics with:
label:
instance_id of firing event
value

Hours to scheduled event


