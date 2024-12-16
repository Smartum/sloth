package prometheusoperator

import (
	"bytes"
	"context"
	"fmt"
	"io"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/prometheus/prometheus/model/rulefmt"
	"github.com/slok/sloth/internal/k8sprometheus"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/slok/sloth/internal/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/types"
)

func NewIOWriterYAMLRepo(writer io.Writer, logger log.Logger) IOWriterPrometheusOperatorYAMLRepo {
	return IOWriterPrometheusOperatorYAMLRepo{
		writer:  writer,
		encoder: json.NewYAMLSerializer(json.DefaultMetaFactory, nil, nil),
		logger:  logger.WithValues(log.Kv{"svc": "storage.IOWriter", "format": "k8s-prometheus-operator"}),
	}
}

// IOWriterPrometheusOperatorYAMLRepo knows to store all the SLO rules (recordings and alerts)
// grouped in an IOWriter in Kubernetes prometheus operator YAML format.
type IOWriterPrometheusOperatorYAMLRepo struct {
	writer  io.Writer
	encoder runtime.Encoder
	logger  log.Logger
}

func (i IOWriterPrometheusOperatorYAMLRepo) StoreSLOs(ctx context.Context, kmeta k8sprometheus.K8sMeta, slos []k8sprometheus.StorageSLO) error {
	rule, err := mapModelToPrometheusOperator(ctx, kmeta, slos)
	if err != nil {
		return fmt.Errorf("could not map model to Prometheus operator CR: %w", err)
	}

	var b bytes.Buffer
	err = i.encoder.Encode(rule, &b)
	if err != nil {
		return fmt.Errorf("could encode prometheus operator object: %w", err)
	}

	rulesYaml := k8sprometheus.WriteTopDisclaimer(b.Bytes())
	_, err = i.writer.Write(rulesYaml)
	if err != nil {
		return fmt.Errorf("could not write top disclaimer: %w", err)
	}

	return nil
}

func mapModelToPrometheusOperator(ctx context.Context, kmeta k8sprometheus.K8sMeta, slos []k8sprometheus.StorageSLO) (*monitoringv1.PrometheusRule, error) {
	// Add extra labels.
	labels := map[string]string{
		"app.kubernetes.io/component":  "SLO",
		"app.kubernetes.io/managed-by": "sloth",
	}
	for k, v := range kmeta.Labels {
		labels[k] = v
	}

	rule := &monitoringv1.PrometheusRule{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "monitoring.coreos.com/v1",
			Kind:       "PrometheusRule",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        kmeta.Name,
			Namespace:   kmeta.Namespace,
			Labels:      labels,
			Annotations: kmeta.Annotations,
		},
	}

	if len(slos) == 0 {
		return nil, fmt.Errorf("slo rules required")
	}

	for _, slo := range slos {
		if len(slo.Rules.SLIErrorRecRules) > 0 {
			rule.Spec.Groups = append(rule.Spec.Groups, monitoringv1.RuleGroup{
				Name:  fmt.Sprintf("sloth-slo-sli-recordings-%s", slo.SLO.ID),
				Rules: promRulesToKubeRules(slo.Rules.SLIErrorRecRules),
			})
		}

		if len(slo.Rules.MetadataRecRules) > 0 {
			rule.Spec.Groups = append(rule.Spec.Groups, monitoringv1.RuleGroup{
				Name:  fmt.Sprintf("sloth-slo-meta-recordings-%s", slo.SLO.ID),
				Rules: promRulesToKubeRules(slo.Rules.MetadataRecRules),
			})
		}

		if len(slo.Rules.AlertRules) > 0 {
			rule.Spec.Groups = append(rule.Spec.Groups, monitoringv1.RuleGroup{
				Name:  fmt.Sprintf("sloth-slo-alerts-%s", slo.SLO.ID),
				Rules: promRulesToKubeRules(slo.Rules.AlertRules),
			})
		}
	}

	// If we don't have anything to store, error so we can increase the reliability
	// because maybe this was due to an unintended error (typos, misconfig, too many disable...).
	if len(rule.Spec.Groups) == 0 {
		return nil, k8sprometheus.ErrNoSLORules
	}

	return rule, nil
}

func promRulesToKubeRules(rules []rulefmt.Rule) []monitoringv1.Rule {
	res := make([]monitoringv1.Rule, 0, len(rules))
	for _, r := range rules {
		forS := ""
		if r.For != 0 {
			forS = r.For.String()
		}

		res = append(res, monitoringv1.Rule{
			Record:      r.Record,
			Alert:       r.Alert,
			Expr:        intstr.FromString(r.Expr),
			For:         forS,
			Labels:      r.Labels,
			Annotations: r.Annotations,
		})
	}
	return res
}

func NewPrometheusOperatorCRDRepo(ensurer k8sprometheus.PrometheusRulesEnsurer, logger log.Logger) CRDRepo {
	return CRDRepo{
		ensurer: ensurer,
		logger:  logger.WithValues(log.Kv{"svc": "storage.PrometheusOperatorCRDAPIServer", "format": "k8s-prometheus-operator"}),
	}
}

// CRDRepo knows to store all the SLO rules (recordings and alerts)
// grouped as a Kubernetes prometheus operator CR using Kubernetes API server.
type CRDRepo struct {
	logger  log.Logger
	ensurer k8sprometheus.PrometheusRulesEnsurer
}

func (p CRDRepo) StoreSLOs(ctx context.Context, kmeta k8sprometheus.K8sMeta, slos []k8sprometheus.StorageSLO) error {
	// Map to the Prometheus operator CRD.
	rule, err := mapModelToPrometheusOperator(ctx, kmeta, slos)
	if err != nil {
		return fmt.Errorf("could not map model to Prometheus operator CR: %w", err)
	}

	// Add object reference.
	rule.ObjectMeta.OwnerReferences = append(rule.ObjectMeta.OwnerReferences, metav1.OwnerReference{
		Kind:       kmeta.Kind,
		APIVersion: kmeta.APIVersion,
		Name:       kmeta.Name,
		UID:        types.UID(kmeta.UID),
	})

	// Create on API server.
	err = p.ensurer.EnsurePrometheusRule(ctx, rule)
	if err != nil {
		return fmt.Errorf("could not ensure Prometheus operator rule CR: %w", err)
	}

	return nil
}