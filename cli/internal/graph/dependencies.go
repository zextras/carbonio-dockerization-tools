package graph

import "github.com/zextras/carbonio-base-dockerization/cli/internal/parser"

// DependencyResolver resolves service dependencies
type DependencyResolver struct {
	services map[string]*parser.ServiceDefinition
}

// NewDependencyResolver creates a new dependency resolver
func NewDependencyResolver(services map[string]*parser.ServiceDefinition) *DependencyResolver {
	return &DependencyResolver{
		services: services,
	}
}

// ResolveDependencies returns all dependencies for a given service (recursive)
func (r *DependencyResolver) ResolveDependencies(serviceName string) []string {
	resolved := make(map[string]bool)
	r.resolveDependenciesRecursive(serviceName, resolved)

	// Convert map to slice
	result := make([]string, 0, len(resolved))
	for name := range resolved {
		if name != serviceName { // Exclude the service itself
			result = append(result, name)
		}
	}

	return result
}

// resolveDependenciesRecursive recursively resolves dependencies
func (r *DependencyResolver) resolveDependenciesRecursive(serviceName string, resolved map[string]bool) {
	// If already resolved, skip
	if resolved[serviceName] {
		return
	}

	resolved[serviceName] = true

	// Get service definition
	svc, exists := r.services[serviceName]
	if !exists {
		return
	}

	// Recursively resolve dependencies
	for _, dep := range svc.DependsOn {
		r.resolveDependenciesRecursive(dep, resolved)
	}
}

// GetAllDependencies returns all dependencies for multiple services
func (r *DependencyResolver) GetAllDependencies(serviceNames []string) []string {
	allDeps := make(map[string]bool)

	for _, name := range serviceNames {
		deps := r.ResolveDependencies(name)
		for _, dep := range deps {
			allDeps[dep] = true
		}
		// Also include the service itself
		allDeps[name] = true
	}

	// Convert to slice
	result := make([]string, 0, len(allDeps))
	for name := range allDeps {
		result = append(result, name)
	}

	return result
}
