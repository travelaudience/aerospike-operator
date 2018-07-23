package versioning

const (
	aerospikeServer_4_2_0_3 = "4.2.0.3"
	aerospikeServer_4_2_0_4 = "4.2.0.4"
)

var (
	// AerospikeServerSupportedVersions holds the list of Aerospike versions
	// currently supported by the operator.
	AerospikeServerSupportedVersions = []string{
		aerospikeServer_4_2_0_3,
		aerospikeServer_4_2_0_4,
	}
)
