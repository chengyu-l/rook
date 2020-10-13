package commons

import (
	"fmt"
	"github.com/rook/rook/pkg/operator/chubao/constants"
)

func GetServiceDomain(serviceName, namespace string) string {
	return fmt.Sprintf("%s.%s.%s", serviceName, namespace, constants.ServiceDomainSuffix)
}
