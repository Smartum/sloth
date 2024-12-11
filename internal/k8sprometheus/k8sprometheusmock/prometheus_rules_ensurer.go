// Code generated by mockery v2.50.0. DO NOT EDIT.

package k8sprometheusmock

import (
	context "context"

	mock "github.com/stretchr/testify/mock"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	v1 "github.com/GoogleCloudPlatform/prometheus-engine/pkg/operator/apis/monitoring/v1"
)

// PrometheusRulesEnsurer is an autogenerated mock type for the PrometheusRulesEnsurer type
type PrometheusRulesEnsurer struct {
	mock.Mock
}

// EnsureManagedPrometheusRule provides a mock function with given fields: ctx, pr
func (_m *PrometheusRulesEnsurer) EnsureManagedPrometheusRule(ctx context.Context, pr *v1.Rules) error {
	ret := _m.Called(ctx, pr)

	if len(ret) == 0 {
		panic("no return value specified for EnsureManagedPrometheusRule")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Rules) error); ok {
		r0 = rf(ctx, pr)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// EnsurePrometheusRule provides a mock function with given fields: ctx, pr
func (_m *PrometheusRulesEnsurer) EnsurePrometheusRule(ctx context.Context, pr *monitoringv1.PrometheusRule) error {
	ret := _m.Called(ctx, pr)

	if len(ret) == 0 {
		panic("no return value specified for EnsurePrometheusRule")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *monitoringv1.PrometheusRule) error); ok {
		r0 = rf(ctx, pr)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewPrometheusRulesEnsurer creates a new instance of PrometheusRulesEnsurer. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewPrometheusRulesEnsurer(t interface {
	mock.TestingT
	Cleanup(func())
}) *PrometheusRulesEnsurer {
	mock := &PrometheusRulesEnsurer{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
