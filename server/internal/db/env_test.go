package db_test

// envVarTests documents the required env vars for each Lambda and provides
// helpers used by handler-level tests to catch missing env vars before deploy.
//
// Pattern used in handler tests:
//
//	func TestHandler_MissingEnvVar_Panics(t *testing.T) {
//	    assertPanicsWithoutEnv(t, "SESSIONS_TABLE", func() {
//	        handler(context.Background(), someValidReq)
//	    })
//	}
//
// This file is intentionally minimal — the real tests live in each cmd/ package.
