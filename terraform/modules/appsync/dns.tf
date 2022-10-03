resource "aws_appsync_domain_name" "regional" {
  domain_name     = "${data.aws_region.current.name}.${var.dns.domain_name}"
  certificate_arn = var.dns.cert
}

resource "aws_appsync_domain_name_api_association" "regional" {
  api_id      = aws_appsync_graphql_api.this.id
  domain_name = aws_appsync_domain_name.regional.domain_name
}

resource "aws_route53_record" "global" {
  zone_id        = var.dns.zone_id
  name           = "${var.dns.domain_name}."
  type           = "CNAME"
  ttl            = 300
  records        = ["${aws_appsync_domain_name.regional.domain_name}."]
  set_identifier = data.aws_region.current.name

  latency_routing_policy {
    region = data.aws_region.current.name
  }

  # todo: health check
}

resource "aws_route53_record" "regional" {
  zone_id = var.dns.zone_id
  name    = "${aws_appsync_domain_name.regional.domain_name}."
  type    = "A"
  alias {
    name                   = aws_appsync_domain_name.regional.appsync_domain_name
    zone_id                = aws_appsync_domain_name.regional.hosted_zone_id
    evaluate_target_health = false
  }
}