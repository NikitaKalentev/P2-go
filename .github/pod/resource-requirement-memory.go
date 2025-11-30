package pod

import (
	"strconv"

	"github.com/NikitaKalentev/P2-go/structure"
)

type ResourceRequirementMemory struct {
	structure.Element
}

func (n *ResourceRequirementMemory) Validate() {
	value := n.GetValue().Value
	if len(value) < 3 {
		n.ErrorInvalidFormat()
		return
	}
	mesurement := value[len(value)-2:]
	if mesurement != "Gi" && mesurement != "Mi" && mesurement != "Ki" {
		n.ErrorInvalidFormat()
		return
	}
	amount := value[0 : len(value)-2]
	if _, err := strconv.Atoi(amount); err != nil {
		n.ErrorInvalidFormat()
	}
}

func NewResourceRequirementMemory() *ResourceRequirementMemory {
	return &ResourceRequirementMemory{}
}