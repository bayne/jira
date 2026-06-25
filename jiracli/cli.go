package jiracli

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"reflect"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"

	"github.com/coryb/figtree"
	"github.com/coryb/oreo"
	"github.com/jinzhu/copier"
	shellquote "github.com/kballard/go-shellquote"
	"github.com/tidwall/gjson"
	"gopkg.in/AlecAivazis/survey.v1"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
	yaml "gopkg.in/coryb/yaml.v2"
	logging "gopkg.in/op/go-logging.v1"
)

type Exit struct {
	Code int
}

// HandleExit will unwind any panics and check to see if they are jiracli.Exit
// and exit accordingly.
//
// Example:
// func main() {
//     defer jiracli.HandleExit()
//     ...
// }
func HandleExit() {
	if e := recover(); e != nil {
		if exit, ok := e.(Exit); ok {
			os.Exit(exit.Code)
		} else {
			fmt.Fprintf(os.Stderr, "%s\n%s", e, debug.Stack())
			os.Exit(1)
		}
	}
}

const (
	ServerDeploymentType = "server"
	CloudDeploymentType  = "cloud"
)

type GlobalOptions struct {
	// AuthenticationMethod is the method we use to authenticate with the jira serivce.
	// Possible values are "api-token", "bearer-token" or "session".
	// The default is "api-token" when the service endpoint ends with "atlassian.net", otherwise it "session".  Session authentication
	// will promt for user password and use the /auth/1/session-login endpoint.
	AuthenticationMethod figtree.StringOption `yaml:"authentication-method,omitempty" json:"authentication-method,omitempty"`

	// Endpoint is the URL for the Jira service.  Something like: https://go-jira.atlassian.net
	Endpoint figtree.StringOption `yaml:"endpoint,omitempty" json:"endpoint,omitempty"`

	// Insecure will allow you to connect to an https endpoint with a self-signed SSL certificate
	Insecure figtree.BoolOption `yaml:"insecure,omitempty" json:"insecure,omitempty"`

	// Login is the id used for authenticating with the Jira service.  For "api-token" AuthenticationMethod this is usually a
	// full email address, something like "user@example.com".  For "session" AuthenticationMethod this will be something
	// like "user", which by default will use the same value in the `User` field.
	Login figtree.StringOption `yaml:"login,omitempty" json:"login,omitempty"`

	// PasswordSource specificies the method that we fetch the password.  Possible values are "keyring" or "pass".
	// If this is unset we will just prompt the user.  For "keyring" this will look in the OS keychain, if missing
	// then prompt the user and store the password in the OS keychain.  For "pass" this will look in the PasswordDirectory
	// location using the `pass` tool, if missing prompt the user and store in the PasswordDirectory
	PasswordSource figtree.StringOption `yaml:"password-source,omitempty" json:"password-source,omitempty"`

	// PasswordSourcePath can be used to specify the path to the PasswordSource binary to use.
	PasswordSourcePath figtree.StringOption `yaml:"password-source-path,omitempty" json:"password-source-path,omitempty"`

	// Cached password to avoid invoking password source on each API request
	cachedPassword string

	// PasswordDirectory is only used for the "pass" PasswordSource.  It is the location for the encrypted password
	// files used by `pass`.  Effectively this overrides the "PASSWORD_STORE_DIR" environment variable
	PasswordDirectory figtree.StringOption `yaml:"password-directory,omitempty" json:"password-directory,omitempty"`

	// PasswordName is the the name of the password key entry stored used with PasswordSource `pass`.
	PasswordName figtree.StringOption `yaml:"password-name,omitempty" json:"password-name,omitempty"`

	// Quiet will lower the defalt log level to suppress the standard output for commands
	Quiet figtree.BoolOption `yaml:"quiet,omitempty" json:"quiet,omitempty"`

	// SocksProxy is used to configure the http client to access the Endpoint via a socks proxy.  The value
	// should be a ip address and port string, something like "127.0.0.1:1080"
	SocksProxy figtree.StringOption `yaml:"socksproxy,omitempty" json:"socksproxy,omitempty"`

	// UnixProxy is use to configure the http client to access the Endpoint via a local unix domain socket used
	// to proxy requests
	UnixProxy figtree.StringOption `yaml:"unixproxy,omitempty" json:"unixproxy,omitempty"`

	// User is use to represent the user on the Jira service.  This can be different from the username used to
	// authenticate with the service.  For example when using AuthenticationMethod `api-token` the Login is
	// typically an email address like `username@example.com` and the User property would be someting like
	// `username`  The User property is used on Jira service API calls that require a user to associate with
	// an Issue (like assigning a Issue to yourself)
	User figtree.StringOption `yaml:"user,omitempty" json:"user,omitempty"`

	// JiraDeploymentType can be `cloud` or `server`, if not set it will be inferred from
	// the /rest/api/2/serverInfo REST API.
	JiraDeploymentType figtree.StringOption `yaml:"jira-deployment-type,omitempty" json:"jira-deployment-type,omitempty"`

	DefaultBoard figtree.StringOption `yaml:"default-board,omitempty" json:"default-board,omitempty"`
	SprintPrefix figtree.StringOption `yaml:"sprint-prefix,omitempty" json:"sprint-prefix,omitempty"`

	// Download will save the command output to files instead of stdout.
	// For single results, saves to a file in the current directory.
	// For list results, creates a subdirectory and fetches/saves each issue.
	Download figtree.BoolOption `yaml:"download,omitempty" json:"download,omitempty"`

	// User-Agent components used to identify this integration on every
	// request per Disney AES REST client guidelines.
	AppName      figtree.StringOption `yaml:"app-name,omitempty" json:"app-name,omitempty"`
	Contact      figtree.StringOption `yaml:"contact,omitempty" json:"contact,omitempty"`
	Runtime      figtree.StringOption `yaml:"runtime,omitempty" json:"runtime,omitempty"`
	Hostname     figtree.StringOption `yaml:"hostname,omitempty" json:"hostname,omitempty"`
	AwsAccountID figtree.StringOption `yaml:"aws-account-id,omitempty" json:"aws-account-id,omitempty"`
	AwsRegion    figtree.StringOption `yaml:"aws-region,omitempty" json:"aws-region,omitempty"`
	Environment  figtree.StringOption `yaml:"environment,omitempty" json:"environment,omitempty"`

	// ValidateToken controls whether each command begins with a probe
	// against /rest/api/2/myself to confirm the PAT or api-token is still
	// accepted. Defaults to false to keep one-shot CLI calls cheap.
	ValidateToken figtree.BoolOption `yaml:"validate-token,omitempty" json:"validate-token,omitempty"`

	// RateLimit caps outbound requests at this many per second using a token
	// bucket, smoothing the cadence of high-volume commands (like crawl) so we
	// trip server rate limits less often. 0 (the default) disables pacing, so
	// one-shot commands are unaffected.
	RateLimit figtree.Float64Option `yaml:"rate-limit,omitempty" json:"rate-limit,omitempty"`

	// RateBurst is the token-bucket burst capacity: how many requests may be
	// sent back-to-back before pacing engages. Only used when RateLimit > 0;
	// defaults to one second's worth of tokens (RateLimit) when unset.
	RateBurst figtree.Float64Option `yaml:"rate-burst,omitempty" json:"rate-burst,omitempty"`
}

type CommonOptions struct {
	Browse      figtree.BoolOption   `yaml:"browse,omitempty" json:"browse,omitempty"`
	Editor      figtree.StringOption `yaml:"editor,omitempty" json:"editor,omitempty"`
	File        figtree.StringOption `yaml:"file,omitempty" json:"file,omitempty"`
	GJsonQuery  figtree.StringOption `yaml:"gjq,omitempty" json:"gjq,omitempty"`
	SkipEditing figtree.BoolOption   `yaml:"noedit,omitempty" json:"noedit,omitempty"`
	Template    figtree.StringOption `yaml:"template,omitempty" json:"template,omitempty"`
}

type CommandRegistryEntry struct {
	Help        string
	UsageFunc   func(*figtree.FigTree, *kingpin.CmdClause) error
	ExecuteFunc func(*oreo.Client, *GlobalOptions) error
}

type CommandRegistry struct {
	Command string
	Aliases []string
	Entry   *CommandRegistryEntry
	Default bool
}

// either kingpin.Application or kingpin.CmdClause fit this interface
type kingpinAppOrCommand interface {
	Command(string, string) *kingpin.CmdClause
	GetCommand(string) *kingpin.CmdClause
}

var globalCommandRegistry = []CommandRegistry{}

func RegisterCommand(regEntry CommandRegistry) {
	globalCommandRegistry = append(globalCommandRegistry, regEntry)
}

func (o *GlobalOptions) AuthMethod() string {
	if strings.Contains(o.Endpoint.Value, ".atlassian.net") && o.AuthenticationMethod.Source == "default" {
		return "api-token"
	}
	return o.AuthenticationMethod.Value
}

func (o *GlobalOptions) AuthMethodIsToken() bool {
	return o.AuthMethod() == "api-token" || o.AuthMethod() == "bearer-token"
}

func register(app *kingpin.Application, o *oreo.Client, fig *figtree.FigTree) {
	globals := GlobalOptions{
		User:                 figtree.NewStringOption(os.Getenv("USER")),
		AuthenticationMethod: figtree.NewStringOption("session"),
	}
	app.Flag("endpoint", "Base URI to use for Jira").Short('e').SetValue(&globals.Endpoint)
	app.Flag("insecure", "Disable TLS certificate verification").Short('k').SetValue(&globals.Insecure)
	app.Flag("quiet", "Suppress output to console").Short('Q').SetValue(&globals.Quiet)
	app.Flag("unixproxy", "Path for a unix-socket proxy").SetValue(&globals.UnixProxy)
	app.Flag("socksproxy", "Address for a socks proxy").SetValue(&globals.SocksProxy)
	app.Flag("user", "user name used within the Jira service").Short('u').SetValue(&globals.User)
	app.Flag("login", "login name that corresponds to the user used for authentication").SetValue(&globals.Login)
	app.Flag("download", "Download results to files in current directory").Short('D').SetValue(&globals.Download)
	app.Flag("app-name", "Application name used in the User-Agent header").SetValue(&globals.AppName)
	app.Flag("contact", "Contact (team DL or addresses) used in the User-Agent header").SetValue(&globals.Contact)
	app.Flag("runtime", "Where this is running (e.g. aws-lambda, aws-ecs, kubernetes, local-dev)").SetValue(&globals.Runtime)
	app.Flag("hostname", "Hostname or worker label used in the User-Agent header").SetValue(&globals.Hostname)
	app.Flag("aws-account-id", "12-digit AWS account ID used in the User-Agent header").SetValue(&globals.AwsAccountID)
	app.Flag("aws-region", "AWS region used in the User-Agent header").SetValue(&globals.AwsRegion)
	app.Flag("environment", "Environment (prod, nonprod, ...) used in the User-Agent header").SetValue(&globals.Environment)
	app.Flag("validate-token", "Probe /rest/api/2/myself before running to confirm the token is still valid").SetValue(&globals.ValidateToken)
	app.Flag("rate-limit", "Cap outbound requests at N per second (0 = unlimited)").SetValue(&globals.RateLimit)
	app.Flag("rate-burst", "Token-bucket burst capacity for --rate-limit (default: one second of tokens)").SetValue(&globals.RateBurst)

	// Centralize retry handling in our post-callback so we can apply
	// exponential backoff with jitter and honor Retry-After uniformly.
	o = o.WithRetries(0)

	// Lazily built on the first request because globals are not populated with
	// flag/config values until after command-line parsing, which happens after
	// these callbacks are registered. nil means rate limiting is disabled.
	var (
		limiterOnce sync.Once
		limiter     *tokenBucket
	)

	o = o.WithPreCallback(func(req *http.Request) (*http.Request, error) {
		limiterOnce.Do(func() {
			if rate := globals.RateLimit.Value; rate > 0 {
				burst := globals.RateBurst.Value
				if burst <= 0 {
					burst = rate
				}
				limiter = newTokenBucket(rate, burst)
			}
		})
		// Acquire a token before doing any work so we pace network egress;
		// retried requests flow through here too and are paced as well.
		if limiter != nil {
			if err := limiter.wait(req.Context()); err != nil {
				return req, err
			}
		}

		// Set (not Add) so retried requests don't accumulate duplicate
		// User-Agent or Authorization headers.
		req.Header.Set("User-Agent", BuildUserAgent(&globals))

		if globals.AuthMethod() == "api-token" {
			token := globals.GetPass()
			authHeader := fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", globals.Login.Value, token))))
			req.Header.Set("Authorization", authHeader)
		} else if globals.AuthMethod() == "bearer-token" {
			token := globals.GetPass()
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		}
		return req, nil
	})

	o = o.WithPostCallback(func(req *http.Request, resp *http.Response) (*http.Response, error) {
		if globals.AuthMethod() == "session" {
			authUser := resp.Header.Get("X-Ausername")
			if authUser == "" || authUser == "anonymous" {
				// preserve the --quiet value, we need to temporarily disable it so
				// the normal login output is surpressed
				defer func(quiet bool) {
					globals.Quiet.Value = quiet
				}(globals.Quiet.Value)
				globals.Quiet.Value = true

				// we are not logged in, so force login now by running the "login" command
				app.Parse([]string{"login"})

				// rerun the original request
				return o.Do(req)
			}
		} else if globals.AuthMethodIsToken() && resp.StatusCode == 401 {
			// AES guidance: do NOT loop on 401 — the token is bad or
			// revoked. Clear the in-memory copy so a subsequent invocation
			// re-reads from the configured source, log an actionable
			// message, and let the caller see the 401.
			globals.ClearCachedPass()
			log.Errorf(
				"%s status=401 method=%s path=%s msg=\"authentication rejected; replace the PAT or api-token (not retrying)\"",
				logContext(&globals), req.Method, req.URL.Path,
			)
			return resp, nil
		}
		return resp, nil
	})

	// Retry callback handles 408/429/504 (and 5xx now that we disabled
	// oreo's built-in retry) with backoff + jitter. We pass a getter so the
	// callback uses the current `o` value (after preActions add transports,
	// proxies, etc.) instead of a stale snapshot.
	o = o.WithPostCallback(retryCallback(func() *oreo.Client { return o }, &globals))

	for _, command := range globalCommandRegistry {
		copy := command
		commandFields := strings.Fields(copy.Command)
		var appOrCmd kingpinAppOrCommand = app
		if len(commandFields) > 1 {
			for _, name := range commandFields[0 : len(commandFields)-1] {
				tmp := appOrCmd.GetCommand(name)
				if tmp == nil {
					tmp = appOrCmd.Command(name, "")
				}
				appOrCmd = tmp
			}
		}

		cmd := appOrCmd.Command(commandFields[len(commandFields)-1], copy.Entry.Help)
		LoadConfigs(cmd, fig, &globals)
		cmd.PreAction(func(_ *kingpin.ParseContext) error {
			if globals.Insecure.Value {
				transport := &http.Transport{
					Proxy: http.ProxyFromEnvironment,
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true,
					},
				}
				o = o.WithTransport(transport)
			}
			if globals.UnixProxy.Value != "" {
				o = o.WithTransport(unixProxy(globals.UnixProxy.Value))
			} else if globals.SocksProxy.Value != "" {
				o = o.WithTransport(socksProxy(globals.SocksProxy.Value))
			}
			if globals.AuthMethodIsToken() {
				o = o.WithCookieFile("")
			}
			if globals.Login.Value == "" {
				globals.Login = globals.User
			}
			if globals.ValidateToken.Value && globals.AuthMethodIsToken() {
				if err := ValidateToken(o, &globals); err != nil {
					log.Errorf("%s", err)
					panic(Exit{Code: 1})
				}
			}
			return nil
		})

		for _, alias := range copy.Aliases {
			cmd = cmd.Alias(alias)
		}
		if copy.Default {
			cmd = cmd.Default()
		}
		if copy.Entry.UsageFunc != nil {
			copy.Entry.UsageFunc(fig, cmd)
		}

		cmd.Action(func(_ *kingpin.ParseContext) error {
			if logging.GetLevel("") > logging.DEBUG {
				o = o.WithTrace(true)
			}
			return copy.Entry.ExecuteFunc(o, &globals)
		})
	}
}

func LoadConfigs(cmd *kingpin.CmdClause, fig *figtree.FigTree, opts interface{}) {
	cmd.PreAction(func(_ *kingpin.ParseContext) error {
		os.Setenv("JIRA_OPERATION", cmd.FullCommand())
		// load command specific configs first
		if err := fig.LoadAllConfigs(strings.Join(strings.Fields(cmd.FullCommand()), "_")+".yml", opts); err != nil {
			return err
		}
		// then load generic configs if not already populated above
		return fig.LoadAllConfigs("config.yml", opts)
	})
}

func BrowseUsage(cmd *kingpin.CmdClause, opts *CommonOptions) {
	cmd.Flag("browse", "Open issue(s) in browser after operation").Short('b').SetValue(&opts.Browse)
}

func EditorUsage(cmd *kingpin.CmdClause, opts *CommonOptions) {
	cmd.Flag("editor", "Editor to use").SetValue(&opts.Editor)
}

func FileUsage(cmd *kingpin.CmdClause, opts *CommonOptions) {
	cmd.Flag("file", "File to use").SetValue(&opts.File)
}

func TemplateUsage(cmd *kingpin.CmdClause, opts *CommonOptions) {
	cmd.Flag("template", "Template to use for output").Short('t').SetValue(&opts.Template)
}

func GJsonQueryUsage(cmd *kingpin.CmdClause, opts *CommonOptions) {
	cmd.Flag("gjq", "GJSON Query to filter output, see https://goo.gl/iaYwJ5").SetValue(&opts.GJsonQuery)
}

func (o *CommonOptions) PrintTemplate(data interface{}) error {
	if o.GJsonQuery.Value != "" {
		buf := bytes.NewBufferString("")
		RunTemplate("json", data, buf)
		results := gjson.GetBytes(buf.Bytes(), o.GJsonQuery.Value)
		_, err := os.Stdout.Write([]byte(results.String()))
		os.Stdout.Write([]byte{'\n'})
		return err
	}
	return RunTemplate(o.Template.Value, data, nil)
}

func (o *CommonOptions) editFile(fileName string) (changes bool, err error) {
	var editor string
	for _, ed := range []string{o.Editor.Value, os.Getenv("JIRA_EDITOR"), os.Getenv("EDITOR"), "vim"} {
		if ed != "" {
			editor = ed
			break
		}
	}

	if o.SkipEditing.Value {
		return false, nil
	}

	tmpFileNameOrig := fmt.Sprintf("%s.orig", fileName)
	if err := copyFile(fileName, tmpFileNameOrig); err != nil {
		return false, err
	}

	defer func() {
		os.Remove(tmpFileNameOrig)
	}()

	shell, _ := shellquote.Split(editor)
	shell = append(shell, fileName)
	log.Debugf("Running: %#v", shell)
	cmd := exec.Command(shell[0], shell[1:]...)
	cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdin
	if err := cmd.Run(); err != nil {
		return false, err
	}

	// now we just need to diff the files to see if there are any changes
	f1, err := os.Open(tmpFileNameOrig)
	if err != nil {
		return false, err
	}
	f2, err := os.Open(fileName)
	if err != nil {
		return false, err
	}

	stat1, err := f1.Stat()
	if err != nil {
		return false, err
	}
	stat2, err := f2.Stat()
	if err != nil {
		return false, err
	}
	// different sizes, so must have changes
	if stat1.Size() != stat2.Size() {
		return true, nil
	}

	p1, p2 := make([]byte, 1024), make([]byte, 1024)
	var n1, n2 int
	// loop though 1024 bytes at a time comparing the buffers for changes
	for err != io.EOF {
		n1, _ = f1.Read(p1)
		n2, err = f2.Read(p2)
		if n1 != n2 {
			return true, nil
		}
		if !bytes.Equal(p1[:n1], p2[:n2]) {
			return true, nil
		}
	}
	return false, nil
}

var EditLoopAbort = fmt.Errorf("edit Loop aborted by request")

func EditLoop(opts *CommonOptions, input interface{}, output interface{}, submit func() error) error {
	tmpFile, err := tmpTemplate(opts.Template.Value, input)
	if err != nil {
		return err
	}

	confirm := func(dflt bool, msg string) (answer bool) {
		survey.AskOne(
			&survey.Confirm{Message: msg, Default: dflt},
			&answer,
			nil,
		)
		return
	}

	// we need to copy the original output so that we can restore
	// it on retries in case we try to populate bogus fields that
	// are rejected by the jira service.
	dup := reflect.New(reflect.ValueOf(output).Elem().Type())
	err = copier.Copy(dup.Interface(), output)
	if err != nil {
		return err
	}

	for {
		if !opts.SkipEditing.Value {
			changes, err := opts.editFile(tmpFile)
			if err != nil {
				log.Error(err.Error())
				if confirm(true, "Editor reported an error, edit again?") {
					continue
				}
				return EditLoopAbort
			}
			if !changes {
				if !confirm(false, "No changes detected, submit anyway?") {
					return EditLoopAbort
				}
			}
		}
		// parse template
		data, err := ioutil.ReadFile(tmpFile)
		if err != nil {
			return err
		}

		defer func(mapType, iface reflect.Type) {
			yaml.DefaultMapType = mapType
			yaml.IfaceType = iface
		}(yaml.DefaultMapType, yaml.IfaceType)
		yaml.DefaultMapType = reflect.TypeOf(map[string]interface{}{})
		yaml.IfaceType = yaml.DefaultMapType.Elem()

		// restore output incase of retry loop
		err = copier.Copy(output, dup.Interface())
		if err != nil {
			return err
		}

		// HACK HACK HACK we want to trim out all the yaml garbage that is not
		// poplulated, like empty arrays, string values with only a newline,
		// etc.  We need to do this because jira will reject json documents
		// with empty arrays, or empty strings typically.  So here we process
		// the data to a raw interface{} then we fixup the yaml parsed
		// interface, then we serialize to a new yaml document ... then is
		// parsed as the original document to populate the output struct.  Phew.
		var raw interface{}
		if err := yaml.Unmarshal(data, &raw); err != nil {
			log.Error(err.Error())
			if confirm(true, "Invalid YAML syntax, edit again?") {
				continue
			}
			return EditLoopAbort
		}
		yamlFixup(&raw)
		fixedYAML, err := yaml.Marshal(&raw)
		if err != nil {
			log.Error(err.Error())
			if confirm(true, "Invalid YAML syntax, edit again?") {
				continue
			}
			return EditLoopAbort
		}

		if err := yaml.Unmarshal(fixedYAML, output); err != nil {
			log.Error(err.Error())
			if confirm(true, "Invalid YAML syntax, edit again?") {
				continue
			}
			return EditLoopAbort
		}
		// submit template
		if err := submit(); err != nil {
			log.Error(err.Error())
			if confirm(true, "Jira reported an error, edit again?") {
				continue
			}
			return EditLoopAbort
		}
		break
	}
	return nil
}

var FileAbort = fmt.Errorf("file processing aborted")

func ReadYmlInputFile(opts *CommonOptions, input interface{}, output interface{}, submit func() error) error {
	tmpFile, err := tmpTemplate(opts.Template.Value, input)
	if err != nil {
		return err
	}

	tmpFile = opts.File.String()

	// we need to copy the original output so that we can restore
	// it on retries in case we try to populate bogus fields that
	// are rejected by the jira service.
	dup := reflect.New(reflect.ValueOf(output).Elem().Type())
	err = copier.Copy(dup.Interface(), output)
	if err != nil {
		return err
	}

	// parse template
	data, err := ioutil.ReadFile(tmpFile)
	if err != nil {
		return err
	}

	defer func(mapType, iface reflect.Type) {
		yaml.DefaultMapType = mapType
		yaml.IfaceType = iface
	}(yaml.DefaultMapType, yaml.IfaceType)
	yaml.DefaultMapType = reflect.TypeOf(map[string]interface{}{})
	yaml.IfaceType = yaml.DefaultMapType.Elem()

	// restore output incase of retry loop
	err = copier.Copy(output, dup.Interface())
	if err != nil {
		return err
	}

	// HACK HACK HACK we want to trim out all the yaml garbage that is not
	// poplulated, like empty arrays, string values with only a newline,
	// etc.  We need to do this because jira will reject json documents
	// with empty arrays, or empty strings typically.  So here we process
	// the data to a raw interface{} then we fixup the yaml parsed
	// interface, then we serialize to a new yaml document ... then is
	// parsed as the original document to populate the output struct.  Phew.
	var raw interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		log.Error(err.Error())
		fmt.Printf("Invalid YAML syntax\n")
		return FileAbort
	}
	yamlFixup(&raw)
	fixedYAML, err := yaml.Marshal(&raw)
	if err != nil {
		log.Error(err.Error())
		fmt.Printf("Invalid YAML syntax\n")
		return FileAbort
	}

	if err := yaml.Unmarshal(fixedYAML, output); err != nil {
		log.Error(err.Error())
		fmt.Printf("Invalid YAML syntax\n")
		return FileAbort
	}
	// submit template
	if err := submit(); err != nil {
		log.Error(err.Error())
		fmt.Printf("Jira reported an error\n")
		return FileAbort
	}
	return nil
}

func FormatIssue(issueKey string, project string) string {
	if issueKey == "" {
		return ""
	}

	// expect PROJ-1234 issue format, this will split and
	// reassemble, converting proj-1234 to PROJ-1234
	parts := strings.SplitN(issueKey, "-", 2)
	if len(parts) > 1 {
		return fmt.Sprintf("%s-%s", strings.ToUpper(parts[0]), parts[1])
	}

	// if issue is not PROJ-1234 then it might just be 1234, so verify
	// it is a number here otherwise warn and return input
	if _, err := strconv.Atoi(issueKey); err != nil {
		log.Warningf("Unexpected issue format %q, expected PROJ-1234", issueKey)
		return issueKey
	}

	if project == "" {
		log.Warningf("Using abbreviated issue %q but `project` property is not defined", issueKey)
		return issueKey
	}

	return fmt.Sprintf("%s-%s", strings.ToUpper(project), issueKey)
}
