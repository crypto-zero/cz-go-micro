package jwt

import (
	"sync"
	"time"

	"c-z.dev/go-micro/auth"
	"c-z.dev/go-micro/auth/rules"
	"c-z.dev/go-micro/auth/token"
)

// NewAuth returns a new instance of the Auth service
func NewAuth(opts ...auth.Option) auth.Auth {
	j := new(jwtAuth)
	j.Init(opts...)
	return j
}

type jwtAuth struct {
	options auth.Options
	jwt     token.Provider
	rules   []*auth.Rule

	sync.Mutex
}

func (j *jwtAuth) String() string {
	return "jwt"
}

func (j *jwtAuth) Init(opts ...auth.Option) {
	j.Lock()
	defer j.Unlock()

	for _, o := range opts {
		o(&j.options)
	}

	j.jwt = NewTokenProvider(
		token.WithPrivateKey(j.options.PrivateKey),
		token.WithPublicKey(j.options.PublicKey),
	)
}

func (j *jwtAuth) Options() auth.Options {
	j.Lock()
	defer j.Unlock()
	return j.options
}

func (j *jwtAuth) Generate(id string, opts ...auth.GenerateOption) (*auth.Account, error) {
	options := auth.NewGenerateOptions(opts...)
	account := &auth.Account{
		ID:       id,
		Type:     options.Type,
		Scopes:   options.Scopes,
		Metadata: options.Metadata,
		Issuer:   j.Options().Namespace,
	}

	// generate a jwtTokenProvider secret which can be provided to the Token() method
	// and exchanged for an access token
	secret, err := j.jwt.Generate(account)
	if err != nil {
		return nil, err
	}
	account.Secret = secret.Token

	// return the account
	return account, nil
}

func (j *jwtAuth) Grant(rule *auth.Rule) error {
	j.Lock()
	defer j.Unlock()
	j.rules = append(j.rules, rule)
	return nil
}

func (j *jwtAuth) Revoke(rule *auth.Rule) error {
	j.Lock()
	defer j.Unlock()

	rules := []*auth.Rule{}
	for _, r := range j.rules {
		if r.ID != rule.ID {
			rules = append(rules, r)
		}
	}

	j.rules = rules
	return nil
}

func (j *jwtAuth) Verify(acc *auth.Account, res *auth.Resource, opts ...auth.VerifyOption) error {
	j.Lock()
	defer j.Unlock()

	var options auth.VerifyOptions
	for _, o := range opts {
		o(&options)
	}

	return rules.Verify(j.rules, acc, res)
}

func (j *jwtAuth) Rules(opts ...auth.RulesOption) ([]*auth.Rule, error) {
	j.Lock()
	defer j.Unlock()
	return j.rules, nil
}

func (j *jwtAuth) Inspect(token string) (*auth.Account, error) {
	return j.jwt.Inspect(token)
}

func (j *jwtAuth) Token(opts ...auth.TokenOption) (*auth.Token, error) {
	options := auth.NewTokenOptions(opts...)

	secret := options.RefreshToken
	if len(options.Secret) > 0 {
		secret = options.Secret
	}

	account, err := j.jwt.Inspect(secret)
	if err != nil {
		return nil, err
	}

	access, err := j.jwt.Generate(account, token.WithExpiry(options.Expiry))
	if err != nil {
		return nil, err
	}

	refresh, err := j.jwt.Generate(account, token.WithExpiry(options.Expiry+time.Hour))
	if err != nil {
		return nil, err
	}

	return &auth.Token{
		Created:      access.Created,
		Expiry:       access.Expiry,
		AccessToken:  access.Token,
		RefreshToken: refresh.Token,
	}, nil
}
