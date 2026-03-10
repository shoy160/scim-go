package util

import (
	"errors"
	"scim-go/model"
)

// ValidatePatchRequest 验证Patch请求
func ValidatePatchRequest(req *model.PatchRequest) error {
	// 校验Patch Schema
	if len(req.Schemas) == 0 || req.Schemas[0] != model.PatchSchema.String() {
		return errors.New("invalid patch schema")
	}

	// 验证操作类型
	validOps := map[string]bool{"add": true, "remove": true, "replace": true}
	for _, op := range req.Operations {
		if !validOps[op.Op] {
			return errors.New("invalid operation: " + op.Op)
		}

		// 当 path 为空时，验证 value 是否存在且为对象类型
		if op.Path == "" {
			if op.Value == nil {
				return errors.New("value is required when path is empty")
			}
			// 验证 value 是否为 map[string]any 类型（对象）
			if _, ok := op.Value.(map[string]any); !ok {
				return errors.New("value must be an object when path is empty")
			}
		}
	}

	return nil
}

// ValidateMemberType 验证成员类型
func ValidateMemberType(memberType string) (model.MemberType, error) {
	if memberType == "" {
		return model.MemberTypeUser, nil
	}

	if mt, ok := model.ParseMemberType(memberType); ok {
		return mt, nil
	}

	return "", errors.New("invalid member type: must be 'User' or 'Group'")
}
