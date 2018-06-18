package versioning

const (
	aerospikeServer_4_0_0_4 = "4.0.0.4"
	aerospikeServer_4_0_0_5 = "4.0.0.5"
	aerospikeServer_4_1_0_1 = "4.1.0.1"
)

var (
	// AerospikeServerSupportedVersions holds the list of Aerospike versions
	// currently supported by the operator.
	AerospikeServerSupportedVersions = []string{
		aerospikeServer_4_0_0_4,
		aerospikeServer_4_0_0_5,
		aerospikeServer_4_1_0_1,
	}
)
