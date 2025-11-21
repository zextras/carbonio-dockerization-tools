package graph

import (
	"carbonio-docker-cli/internal/parser"
)

type DependencyResolver struct {
	services map[string]*parser.ServiceDefinition
}

func NewDependencyResolver(services map[string]*parser.ServiceDefinition) *DependencyResolver {
	return &DependencyResolver{
		services: services,
	}
}
func (r *DependencyResolver) ResolveDependencies(serviceName string) []string {
	resolved := make(map[string]bool)
	r.resolveDependenciesRecursive(serviceName, resolved)
	result := make([]string, 0, len(resolved))
	for name := range resolved {
		if name != serviceName {
			result = append(result, name)
		}
	}
	return result
}
func (r *DependencyResolver) resolveDependenciesRecursive(serviceName string, resolved map[string]bool) {
	if resolved[serviceName] {
		return
	}
	resolved[serviceName] = true
	svc, exists := r.services[serviceName]
	if !exists {
		return
	}
	for _, dep := range svc.DependsOn {
		r.resolveDependenciesRecursive(dep, resolved)
	}
}
func (r *DependencyResolver) GetAllDependencies(serviceNames []string) []string {
	allDeps := make(map[string]bool)
	for _, name := range serviceNames {
		deps := r.ResolveDependencies(name)
		for _, dep := range deps {
			allDeps[dep] = true
		}
		allDeps[name] = true
	}
	result := make([]string, 0, len(allDeps))
	for name := range allDeps {
		result = append(result, name)
	}
	return result
}
