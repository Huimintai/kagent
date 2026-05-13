/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package platform

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/kagent-dev/kagent/go/api/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/spf13/cobra"
)

// Supported platforms for the connect command.
var supportedPlatforms = []string{"github", "jira", "slack", "k8s"}

// ConnectOptions holds configuration for the platform connect command.
type ConnectOptions struct {
	Platform  string
	Namespace string
	Name      string
}

// NewPlatformCmd creates the root platform command with all subcommands.
func NewPlatformCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "platform",
		Short: "Platform credential management",
		Long:  `Commands for connecting external platforms and managing platform credentials.`,
	}

	cmd.AddCommand(newConnectCmd())
	return cmd
}

func newConnectCmd() *cobra.Command {
	opts := &ConnectOptions{
		Namespace: "kagent",
	}

	cmd := &cobra.Command{
		Use:   "connect <platform>",
		Short: "Connect an external platform by storing credentials",
		Long: `Connect an external platform by authenticating and storing credentials
as a PlatformCredential CRD in the cluster.

Supported platforms:
  github  - GitHub App credentials (App ID, Installation ID, private key)
  jira    - Atlassian Jira via OAuth2 flow
  slack   - Slack bot token or OAuth2 flow
  k8s     - Kubernetes ServiceAccount reference

Examples:
  kagent platform connect github
  kagent platform connect jira --name my-jira-cred
  kagent platform connect slack --namespace my-ns
  kagent platform connect k8s`,
		Args: cobra.ExactArgs(1),
		ValidArgs: supportedPlatforms,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Platform = args[0]
			return runConnect(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Namespace, "namespace", "n", "kagent", "Namespace for the PlatformCredential")
	cmd.Flags().StringVar(&opts.Name, "name", "", "Name for the PlatformCredential (defaults to <platform>-default)")

	return cmd
}

func runConnect(ctx context.Context, opts *ConnectOptions) error {
	if opts.Name == "" {
		opts.Name = opts.Platform + "-default"
	}

	k8sClient, err := createK8sClient()
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	switch opts.Platform {
	case "github":
		return connectGitHub(ctx, k8sClient, opts)
	case "jira":
		return connectJira(ctx, k8sClient, opts)
	case "slack":
		return connectSlack(ctx, k8sClient, opts)
	case "k8s":
		return connectK8s(ctx, k8sClient, opts)
	default:
		return fmt.Errorf("unsupported platform: %s (supported: %s)", opts.Platform, strings.Join(supportedPlatforms, ", "))
	}
}

// connectGitHub prompts for GitHub App credentials and creates the PlatformCredential.
func connectGitHub(ctx context.Context, k8sClient client.Client, opts *ConnectOptions) error {
	fmt.Println("Connecting GitHub App credentials...")
	fmt.Println()

	appID, err := promptInput("GitHub App ID: ")
	if err != nil {
		return fmt.Errorf("failed to read App ID: %w", err)
	}

	installationID, err := promptInput("Installation ID: ")
	if err != nil {
		return fmt.Errorf("failed to read Installation ID: %w", err)
	}

	privateKeyPath, err := promptInput("Path to private key file (.pem): ")
	if err != nil {
		return fmt.Errorf("failed to read private key path: %w", err)
	}

	privateKeyBytes, err := os.ReadFile(strings.TrimSpace(privateKeyPath))
	if err != nil {
		return fmt.Errorf("failed to read private key file: %w", err)
	}

	secretData := map[string][]byte{
		"appID":          []byte(strings.TrimSpace(appID)),
		"installationID": []byte(strings.TrimSpace(installationID)),
		"privateKey":     privateKeyBytes,
	}

	secretName := opts.Name + "-secret"
	if err := createOrUpdateSecretFromData(ctx, k8sClient, opts.Namespace, secretName, secretData); err != nil {
		return fmt.Errorf("failed to create secret: %w", err)
	}

	if err := createOrUpdatePlatformCredential(ctx, k8sClient, opts, secretName, v1alpha2.CredentialSourceTypeGitHubApp, ""); err != nil {
		return fmt.Errorf("failed to create PlatformCredential: %w", err)
	}

	fmt.Printf("\nPlatformCredential '%s' created in namespace '%s'\n", opts.Name, opts.Namespace)
	return nil
}

// connectJira performs an OAuth2 flow for Atlassian Jira.
func connectJira(ctx context.Context, k8sClient client.Client, opts *ConnectOptions) error {
	fmt.Println("Connecting Jira via OAuth2...")
	fmt.Println()

	clientID, err := promptInput("Atlassian OAuth2 Client ID: ")
	if err != nil {
		return fmt.Errorf("failed to read client ID: %w", err)
	}

	clientSecret, err := promptInput("Atlassian OAuth2 Client Secret: ")
	if err != nil {
		return fmt.Errorf("failed to read client secret: %w", err)
	}

	flow := &OAuthFlow{
		AuthURL:      "https://auth.atlassian.com/authorize",
		TokenURL:     "https://auth.atlassian.com/oauth/token",
		ClientID:     strings.TrimSpace(clientID),
		ClientSecret: strings.TrimSpace(clientSecret),
		Scopes:       []string{"read:jira-work", "write:jira-work", "read:jira-user", "offline_access"},
	}

	result, err := flow.Run(ctx)
	if err != nil {
		return fmt.Errorf("OAuth2 flow failed: %w", err)
	}

	secretData := map[string][]byte{
		"clientID":     []byte(strings.TrimSpace(clientID)),
		"clientSecret": []byte(strings.TrimSpace(clientSecret)),
		"accessToken":  []byte(result.AccessToken),
		"refreshToken": []byte(result.RefreshToken),
	}

	secretName := opts.Name + "-secret"
	if err := createOrUpdateSecretFromData(ctx, k8sClient, opts.Namespace, secretName, secretData); err != nil {
		return fmt.Errorf("failed to create secret: %w", err)
	}

	if err := createOrUpdatePlatformCredential(ctx, k8sClient, opts, secretName, v1alpha2.CredentialSourceTypeOAuth2, "https://auth.atlassian.com/oauth/token"); err != nil {
		return fmt.Errorf("failed to create PlatformCredential: %w", err)
	}

	fmt.Printf("\nPlatformCredential '%s' created in namespace '%s'\n", opts.Name, opts.Namespace)
	return nil
}

// connectSlack prompts for a Slack bot token or initiates an OAuth2 flow.
func connectSlack(ctx context.Context, k8sClient client.Client, opts *ConnectOptions) error {
	fmt.Println("Connecting Slack...")
	fmt.Println()
	fmt.Println("Choose authentication method:")
	fmt.Println("  1) Bot Token (paste an xoxb-... token)")
	fmt.Println("  2) OAuth2 flow (requires Client ID and Secret)")
	fmt.Println()

	choice, err := promptInput("Selection [1/2]: ")
	if err != nil {
		return fmt.Errorf("failed to read selection: %w", err)
	}

	switch strings.TrimSpace(choice) {
	case "1":
		return connectSlackBotToken(ctx, k8sClient, opts)
	case "2":
		return connectSlackOAuth(ctx, k8sClient, opts)
	default:
		return fmt.Errorf("invalid selection: %s (choose 1 or 2)", choice)
	}
}

func connectSlackBotToken(ctx context.Context, k8sClient client.Client, opts *ConnectOptions) error {
	token, err := promptInput("Slack Bot Token (xoxb-...): ")
	if err != nil {
		return fmt.Errorf("failed to read token: %w", err)
	}

	token = strings.TrimSpace(token)
	if !strings.HasPrefix(token, "xoxb-") {
		fmt.Println("Warning: token does not start with 'xoxb-'. Are you sure this is a bot token?")
	}

	secretData := map[string][]byte{
		"botToken": []byte(token),
	}

	secretName := opts.Name + "-secret"
	if err := createOrUpdateSecretFromData(ctx, k8sClient, opts.Namespace, secretName, secretData); err != nil {
		return fmt.Errorf("failed to create secret: %w", err)
	}

	if err := createOrUpdatePlatformCredential(ctx, k8sClient, opts, secretName, v1alpha2.CredentialSourceTypeBotToken, ""); err != nil {
		return fmt.Errorf("failed to create PlatformCredential: %w", err)
	}

	fmt.Printf("\nPlatformCredential '%s' created in namespace '%s'\n", opts.Name, opts.Namespace)
	return nil
}

func connectSlackOAuth(ctx context.Context, k8sClient client.Client, opts *ConnectOptions) error {
	clientID, err := promptInput("Slack OAuth2 Client ID: ")
	if err != nil {
		return fmt.Errorf("failed to read client ID: %w", err)
	}

	clientSecret, err := promptInput("Slack OAuth2 Client Secret: ")
	if err != nil {
		return fmt.Errorf("failed to read client secret: %w", err)
	}

	flow := &OAuthFlow{
		AuthURL:      "https://slack.com/oauth/v2/authorize",
		TokenURL:     "https://slack.com/api/oauth.v2.access",
		ClientID:     strings.TrimSpace(clientID),
		ClientSecret: strings.TrimSpace(clientSecret),
		Scopes:       []string{"chat:write", "channels:read", "channels:history", "users:read"},
	}

	result, err := flow.Run(ctx)
	if err != nil {
		return fmt.Errorf("OAuth2 flow failed: %w", err)
	}

	secretData := map[string][]byte{
		"clientID":     []byte(strings.TrimSpace(clientID)),
		"clientSecret": []byte(strings.TrimSpace(clientSecret)),
		"accessToken":  []byte(result.AccessToken),
		"refreshToken": []byte(result.RefreshToken),
	}

	secretName := opts.Name + "-secret"
	if err := createOrUpdateSecretFromData(ctx, k8sClient, opts.Namespace, secretName, secretData); err != nil {
		return fmt.Errorf("failed to create secret: %w", err)
	}

	if err := createOrUpdatePlatformCredential(ctx, k8sClient, opts, secretName, v1alpha2.CredentialSourceTypeOAuth2, "https://slack.com/api/oauth.v2.access"); err != nil {
		return fmt.Errorf("failed to create PlatformCredential: %w", err)
	}

	fmt.Printf("\nPlatformCredential '%s' created in namespace '%s'\n", opts.Name, opts.Namespace)
	return nil
}

// connectK8s prompts for a ServiceAccount reference.
func connectK8s(ctx context.Context, k8sClient client.Client, opts *ConnectOptions) error {
	fmt.Println("Connecting Kubernetes ServiceAccount...")
	fmt.Println()

	saName, err := promptInput("ServiceAccount name: ")
	if err != nil {
		return fmt.Errorf("failed to read ServiceAccount name: %w", err)
	}

	saNamespace, err := promptInput(fmt.Sprintf("ServiceAccount namespace [%s]: ", opts.Namespace))
	if err != nil {
		return fmt.Errorf("failed to read ServiceAccount namespace: %w", err)
	}
	saNamespace = strings.TrimSpace(saNamespace)
	if saNamespace == "" {
		saNamespace = opts.Namespace
	}

	secretData := map[string][]byte{
		"serviceAccountName":      []byte(strings.TrimSpace(saName)),
		"serviceAccountNamespace": []byte(saNamespace),
	}

	secretName := opts.Name + "-secret"
	if err := createOrUpdateSecretFromData(ctx, k8sClient, opts.Namespace, secretName, secretData); err != nil {
		return fmt.Errorf("failed to create secret: %w", err)
	}

	if err := createOrUpdatePlatformCredential(ctx, k8sClient, opts, secretName, v1alpha2.CredentialSourceTypeServiceAccount, ""); err != nil {
		return fmt.Errorf("failed to create PlatformCredential: %w", err)
	}

	fmt.Printf("\nPlatformCredential '%s' created in namespace '%s'\n", opts.Name, opts.Namespace)
	return nil
}

// createOrUpdateSecretFromData creates or updates a Kubernetes Secret with the given data.
func createOrUpdateSecretFromData(ctx context.Context, k8sClient client.Client, namespace, name string, data map[string][]byte) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "kagent-cli",
				"kagent.dev/credential":        "true",
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: data,
	}

	existing := &corev1.Secret{}
	err := k8sClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, existing)
	if err != nil {
		if apierrors.IsNotFound(err) {
			if err := k8sClient.Create(ctx, secret); err != nil {
				return fmt.Errorf("failed to create secret '%s': %w", name, err)
			}
			fmt.Printf("Created secret '%s' in namespace '%s'\n", name, namespace)
			return nil
		}
		return fmt.Errorf("failed to check for existing secret: %w", err)
	}

	existing.Data = data
	existing.Labels = secret.Labels
	if err := k8sClient.Update(ctx, existing); err != nil {
		return fmt.Errorf("failed to update secret '%s': %w", name, err)
	}
	fmt.Printf("Updated secret '%s' in namespace '%s'\n", name, namespace)
	return nil
}

// createOrUpdatePlatformCredential creates or updates a PlatformCredential CRD.
func createOrUpdatePlatformCredential(ctx context.Context, k8sClient client.Client, opts *ConnectOptions, secretName string, sourceType v1alpha2.CredentialSourceType, tokenEndpoint string) error {
	pcred := &v1alpha2.PlatformCredential{
		ObjectMeta: metav1.ObjectMeta{
			Name:      opts.Name,
			Namespace: opts.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "kagent-cli",
			},
		},
		Spec: v1alpha2.PlatformCredentialSpec{
			Platform: opts.Platform,
			Source: v1alpha2.CredentialSource{
				Type: sourceType,
				SecretRef: &v1alpha2.SecretRef{
					Name: secretName,
				},
				TokenEndpoint: tokenEndpoint,
			},
			AccessPolicy: []v1alpha2.AccessPolicyRule{
				{
					Principals: []string{"agent:*"},
					Scopes:     []string{"*"},
				},
			},
		},
	}
	pcred.SetGroupVersionKind(v1alpha2.GroupVersion.WithKind("PlatformCredential"))

	existing := &v1alpha2.PlatformCredential{}
	err := k8sClient.Get(ctx, client.ObjectKey{Namespace: opts.Namespace, Name: opts.Name}, existing)
	if err != nil {
		if apierrors.IsNotFound(err) {
			if err := k8sClient.Create(ctx, pcred); err != nil {
				return fmt.Errorf("failed to create PlatformCredential '%s': %w", opts.Name, err)
			}
			return nil
		}
		return fmt.Errorf("failed to check for existing PlatformCredential: %w", err)
	}

	existing.Spec = pcred.Spec
	existing.Labels = pcred.Labels
	if err := k8sClient.Update(ctx, existing); err != nil {
		return fmt.Errorf("failed to update PlatformCredential '%s': %w", opts.Name, err)
	}
	return nil
}

// createK8sClient creates a controller-runtime client configured from kubeconfig.
func createK8sClient() (client.Client, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get Kubernetes config: %w", err)
	}

	schemes := runtime.NewScheme()
	if err := scheme.AddToScheme(schemes); err != nil {
		return nil, fmt.Errorf("failed to add core scheme: %w", err)
	}
	if err := v1alpha2.AddToScheme(schemes); err != nil {
		return nil, fmt.Errorf("failed to add v1alpha2 scheme: %w", err)
	}

	k8sClient, err := client.New(restConfig, client.Options{Scheme: schemes})
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	return k8sClient, nil
}

// promptInput prints a prompt and reads a line of input from stdin.
func promptInput(prompt string) (string, error) {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(input, "\n\r"), nil
}
