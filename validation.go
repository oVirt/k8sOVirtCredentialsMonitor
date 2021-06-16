package k8sOVirtCredentialsMonitor

func ValidateCredentials(connection OVirtConnection) error {
	return connection.GetSDK().Test()
}
