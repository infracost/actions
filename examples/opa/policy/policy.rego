package infracost

# totalDiff
deny[msg] {
	maxDiff = 1500.0
	to_number(input.diffTotalMonthlyCost) >= maxDiff

	msg := sprintf(
		"Total monthly cost diff must be less than $%.2f (actual diff is $%.2f)",
		[maxDiff, to_number(input.diffTotalMonthlyCost)],
	)
}

# instanceCost
deny[msg] {
	r := input.projects[_].breakdown.resources[_]
	startswith(r.name, "aws_instance.")

	maxHourlyCost := 2.0
	to_number(r.hourlyCost) > maxHourlyCost

	msg := sprintf(
		"AWS instances must cost less than $%.2f\\hr (%s costs $%.2f\\hr).",
		[maxHourlyCost, r.name, to_number(r.hourlyCost)],
	)
}

# instanceIOPSCost
deny[msg] {
	r := input.projects[_].breakdown.resources[_]
	startswith(r.name, "aws_instance.")

	baseHourlyCost := to_number(r.costComponents[_].hourlyCost)

	sr_cc := r.subresources[_].costComponents[_]
	sr_cc.name == "Provisioned IOPS"
	iopsHourlyCost := to_number(sr_cc.hourlyCost)

	iopsHourlyCost > baseHourlyCost

	msg := sprintf(
		"AWS instance IOPS must cost less than compute usage (%s IOPS $%.2f\\hr, usage $%.2f\\hr).",
		[r.name, iopsHourlyCost, baseHourlyCost],
	)
}
