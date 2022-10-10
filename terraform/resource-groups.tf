locals {
  resource_group_query = {
    ResourceTypeFilters = ["AWS::AllSupported"]
    TagFilters          = [{ Key = "project", Values = [var.name] }]
  }
}

resource "aws_resourcegroups_group" "us_east_1" {
  name = var.name

  resource_query {
    type  = "TAG_FILTERS_1_0"
    query = jsonencode(local.resource_group_query)
  }
}

resource "aws_resourcegroups_group" "us_east_2" {
  provider = aws.us_east_2
  name     = var.name

  resource_query {
    type  = "TAG_FILTERS_1_0"
    query = jsonencode(local.resource_group_query)
  }
}