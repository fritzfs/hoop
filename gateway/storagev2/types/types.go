package types

import (
	"encoding/json"
	"time"

	"olympos.io/encoding/edn"
)

type APIContext struct {
	OrgID          string           `json:"org_id"`
	OrgName        string           `json:"org_name"`
	OrgLicense     string           `json:"org_license"`
	OrgLicenseData *json.RawMessage `json:"org_license_data"`
	UserID         string           `json:"user_id"`
	UserName       string           `json:"user_name"`
	UserEmail      string           `json:"user_email"`
	UserGroups     []string         `json:"user_groups"`
	UserStatus     string           `json:"user_status"`
	SlackID        string           `json:"slack_id"`
	UserPicture    string           `json:"picture"`

	UserAnonSubject       string
	UserAnonEmail         string
	UserAnonProfile       string
	UserAnonPicture       string
	UserAnonEmailVerified *bool

	ApiURL  string `json:"-"`
	GrpcURL string `json:"-"`
}

type Plugin struct {
	ID             string              `json:"id"          edn:"xt/id"`
	OrgID          string              `json:"-"           edn:"plugin/org"`
	Name           string              `json:"name"        edn:"plugin/name"`
	Connections    []*PluginConnection `json:"connections" edn:"plugin/connections,omitempty"`
	ConnectionsIDs []string            `json:"-"           edn:"plugin/connection-ids"`
	Config         *PluginConfig       `json:"config"      edn:"plugin/config,omitempty"`
	ConfigID       *string             `json:"-"           edn:"plugin/config-id"`
	Source         *string             `json:"source"      edn:"plugin/source"`
	Priority       int                 `json:"priority"    edn:"plugin/priority"`
	InstalledById  string              `json:"-"           edn:"plugin/installed-by"`
}

type PluginConfig struct {
	ID      string            `json:"id"      edn:"xt/id"`
	OrgID   string            `json:"-"       edn:"pluginconfig/org"`
	EnvVars map[string]string `json:"envvars" edn:"pluginconfig/envvars"`
}

type PluginConnection struct {
	ID           string   `json:"-"      edn:"xt/id"`
	ConnectionID string   `json:"id"     edn:"plugin-connection/id"`
	Name         string   `json:"name"   edn:"plugin-connection/name"`
	Config       []string `json:"config" edn:"plugin-connection/config"`

	Connection Connection `json:"-" edn:"connection,omitempty"`
}

type Login struct {
	ID       string `edn:"xt/id"`
	Redirect string `edn:"login/redirect"`
	Outcome  string `edn:"login/outcome"`
	SlackID  string `edn:"login/slack-id"`
}

type Client struct {
	ID                    string            `edn:"xt/id"`
	OrgID                 string            `edn:"client/org"`
	Status                ClientStatusType  `edn:"client/status"`
	RequestConnectionName string            `edn:"client/request-connection"`
	RequestPort           string            `edn:"client/request-port"`
	RequestAccessDuration time.Duration     `edn:"client/access-duration"`
	ClientMetadata        map[string]string `edn:"client/metadata"`
	ConnectedAt           time.Time         `edn:"client/connected-at"`
}

type Connection struct {
	Id                 string         `json:"id"`
	OrgId              string         `json:"-"`
	Name               string         `json:"name"`
	IconName           string         `json:"icon_name"`
	Command            []string       `json:"command"`
	Type               string         `json:"type"`
	SubType            string         `json:"subtype"`
	Secret             map[string]any `json:"secret"`
	AgentId            string         `json:"agent_id"`
	AccessModeRunbooks string         `json:"access_mode_runbooks"`
	AccessModeExec     string         `json:"access_mode_exec"`
	AccessModeConnect  string         `json:"access_mode_connect"`
	AccessSchema       string         `json:"access_schema"`
}

type ConnectionInfo struct {
	ID                               string
	Name                             string
	Type                             string
	SubType                          string
	CmdEntrypoint                    []string
	Secrets                          map[string]any
	AgentID                          string
	AgentName                        string
	AgentMode                        string
	AccessModeRunbooks               string
	AccessModeExec                   string
	AccessModeConnect                string
	AccessSchema                     string
	JiraTransitionNameOnSessionClose string
}

type ReviewOwner struct {
	Id      string `json:"id,omitempty"   edn:"xt/id"`
	Name    string `json:"name,omitempty" edn:"review-user/name"`
	Email   string `json:"email"          edn:"review-user/email"`
	SlackID string `json:"slack_id"       edn:"review-user/slack-id"`
}

type ReviewConnection struct {
	Id   string `json:"id,omitempty" edn:"xt/id"`
	Name string `json:"name"         edn:"review-connection/name"`
}

type ReviewGroup struct {
	Id         string       `json:"id"          edn:"xt/id"`
	Group      string       `json:"group"       edn:"review-group/group"`
	Status     ReviewStatus `json:"status"      edn:"review-group/status"`
	ReviewedBy *ReviewOwner `json:"reviewed_by" edn:"review-group/reviewed-by"`
	ReviewDate *string      `json:"review_date" edn:"review-group/review_date"`
}

type Review struct {
	Id               string            `edn:"xt/id"`
	OrgId            string            `edn:"review/org"`
	CreatedAt        time.Time         `edn:"review/created-at"`
	Type             string            `edn:"review/type"`
	Session          string            `edn:"review/session"`
	Input            string            `edn:"review/input"`
	InputEnvVars     map[string]string `edn:"review/input-envvars"`
	InputClientArgs  []string          `edn:"review/input-clientargs"`
	AccessDuration   time.Duration     `edn:"review/access-duration"`
	Status           ReviewStatus      `edn:"review/status"`
	RevokeAt         *time.Time        `edn:"review/revoke-at"`
	CreatedBy        any               `edn:"review/created-by"`
	ReviewOwner      ReviewOwner       `edn:"review/review-owner"`
	ConnectionId     any               `edn:"review/connection"`
	Connection       ReviewConnection  `edn:"review/review-connection"`
	ReviewGroupsIds  []string          `edn:"review/review-groups"`
	ReviewGroupsData []ReviewGroup     `edn:"review/review-groups-data"`
}

type ReviewJSON struct {
	Id        string    `json:"id"`
	OrgId     string    `json:"org"`
	CreatedAt time.Time `json:"created_at"`
	Type      string    `json:"type"`
	Session   string    `json:"session"`
	Input     string    `json:"input"`
	// Redacted for now
	// InputEnvVars     map[string]string `json:"input_envvars"`
	InputClientArgs  []string         `json:"input_clientargs"`
	AccessDuration   time.Duration    `json:"access_duration"`
	Status           ReviewStatus     `json:"status"`
	RevokeAt         *time.Time       `json:"revoke_at"`
	ReviewOwner      ReviewOwner      `json:"review_owner"`
	Connection       ReviewConnection `json:"review_connection"`
	ReviewGroupsData []ReviewGroup    `json:"review_groups_data"`
}

type SessionEventStream []any
type SessionNonIndexedEventStreamList map[edn.Keyword][]SessionEventStream
type SessionScript map[edn.Keyword]string
type SessionLabels map[string]string

type Session struct {
	ID                   string             `json:"id"`
	OrgID                string             `json:"org_id"`
	Script               SessionScript      `json:"script"`
	Labels               SessionLabels      `json:"labels"`
	IntegrationsMetadata map[string]any     `json:"integrations_metadata"`
	Metadata             map[string]any     `json:"metadata"`
	Metrics              map[string]any     `json:"metrics"`
	UserEmail            string             `json:"user"`
	UserID               string             `json:"user_id"`
	UserName             string             `json:"user_name"`
	Type                 string             `json:"type"`
	Connection           string             `json:"connection"`
	Review               *ReviewJSON        `json:"review"`
	Verb                 string             `json:"verb"`
	Status               string             `json:"status"`
	EventStream          SessionEventStream `json:"event_stream"`
	// Must NOT index streams (all top keys are indexed in xtdb)
	NonIndexedStream SessionNonIndexedEventStreamList `json:"-"`
	EventSize        int64                            `json:"event_size"`
	StartSession     time.Time                        `json:"start_date"`
	EndSession       *time.Time                       `json:"end_date"`
	ExitCode         *int                             `json:"exit_code"`
}
