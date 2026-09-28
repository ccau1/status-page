module status-page/packages/api

go 1.25.0

require (
	status-page/packages/core v0.0.0
	status-page/packages/auth v0.0.0
)

replace status-page/packages/core => ../core
replace status-page/packages/auth => ../auth
