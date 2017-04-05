package keepalivedcp

import "testing"

func TestAllocateIP(t *testing.T) {
	type testDef struct {
		name       string
		config     config
		cidr       string
		expectedIP string
		err        bool
	}

	tests := []testDef{
		{
			name: "allocate ip address in empty pool",
			config: config{
				Services: []serviceConfig{},
			},
			cidr:       "10.0.0.0/8",
			expectedIP: "10.0.0.1",
		},
		{
			name: "allocate ip address in pool with one address",
			config: config{
				Services: []serviceConfig{
					{
						UID: "a",
						IP:  "10.0.0.1",
					},
				},
			},
			cidr:       "10.0.0.0/8",
			expectedIP: "10.0.0.2",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(test testDef) func(*testing.T) {
			return func(t *testing.T) {
				ip, err := test.config.allocateIP(test.cidr)

				if err != nil {
					if test.err {
						return
					}

					t.Errorf("got error: %s", err.Error())
					return
				}

				if ip != test.expectedIP {
					t.Errorf("expected IP '%s' but got '%s'", test.expectedIP, ip)
				}
			}
		}(test))
	}
}
