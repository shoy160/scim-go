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
