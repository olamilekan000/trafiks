package models

import (
	"time"

	"github.com/trafiks/trafiks/pkg"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type LogEvent string

const (
	// User Events
	UserSignupSuccess       LogEvent = "user.signup.success"
	UserSignupFailed        LogEvent = "user.signup.failed"
	UserLoginSuccess        LogEvent = "user.login.success"
	UserLoginFailed         LogEvent = "user.login.failed"
	UserLogoutSuccess       LogEvent = "user.logout.success"
	UserLogoutFailed        LogEvent = "user.logout.failed"
	UserVerificationSuccess LogEvent = "user.verification.success"
	UserVerificationFailed  LogEvent = "user.verification.failed"
	PasswordResetRequest    LogEvent = "user.password.reset.request"
	PasswordResetSuccess    LogEvent = "user.password.reset.success"
	PasswordChangeSuccess   LogEvent = "user.password.change.success"
	PasswordChangeFailed    LogEvent = "user.password.change.failed"

	// Organization Events
	OrganizationCreateSuccess LogEvent = "organization.create.success"
	OrganizationCreateFailed  LogEvent = "organization.create.failed"
	OrganizationUpdateSuccess LogEvent = "organization.update.success"
	OrganizationUpdateFailed  LogEvent = "organization.update.failed"
	OrganizationDeleteSuccess LogEvent = "organization.delete.success"
	OrganizationDeleteFailed  LogEvent = "organization.delete.failed"
	OrganizationView          LogEvent = "organization.view"

	// TIN Verification Events
	TINVerificationSuccess LogEvent = "tin.verification.success"
	TINVerificationFailed  LogEvent = "tin.verification.failed"
	TINVerificationCached  LogEvent = "tin.verification.cached"

	// Invoice Events
	InvoiceCreateSuccess    LogEvent = "invoice.create.success"
	InvoiceCreateFailed     LogEvent = "invoice.create.failed"
	InvoiceUpdateSuccess    LogEvent = "invoice.update.success"
	InvoiceUpdateFailed     LogEvent = "invoice.update.failed"
	InvoiceDeleteSuccess    LogEvent = "invoice.delete.success"
	InvoiceDeleteFailed     LogEvent = "invoice.delete.failed"
	InvoiceSubmitFIRS       LogEvent = "invoice.submit.firs"
	InvoiceSubmitFIRSFailed LogEvent = "invoice.submit.firs.failed"
	InvoiceView             LogEvent = "invoice.view"

	// Invoice Line Item Events
	InvoiceLineItemCreateSuccess LogEvent = "invoice.lineitem.create.success"
	InvoiceLineItemUpdateSuccess LogEvent = "invoice.lineitem.update.success"
	InvoiceLineItemDeleteSuccess LogEvent = "invoice.lineitem.delete.success"

	// Product Events
	ProductCreateSuccess LogEvent = "product.create.success"
	ProductCreateFailed  LogEvent = "product.create.failed"
	ProductUpdateSuccess LogEvent = "product.update.success"
	ProductUpdateFailed  LogEvent = "product.update.failed"
	ProductDeleteSuccess LogEvent = "product.delete.success"
	ProductDeleteFailed  LogEvent = "product.delete.failed"
	ProductView          LogEvent = "product.view"

	// Product Category Events
	ProductCategoryCreateSuccess LogEvent = "product.category.create.success"
	ProductCategoryCreateFailed  LogEvent = "product.category.create.failed"
	ProductCategoryUpdateSuccess LogEvent = "product.category.update.success"
	ProductCategoryUpdateFailed  LogEvent = "product.category.update.failed"
	ProductCategoryDeleteSuccess LogEvent = "product.category.delete.success"
	ProductCategoryDeleteFailed  LogEvent = "product.category.delete.failed"
)

type LogSeverity string

const (
	LogError   LogSeverity = "error"
	LogInfo    LogSeverity = "info"
	LogWarning LogSeverity = "warning"
)

type ResourceCategory string

const (
	CategoryAuthentication  ResourceCategory = "authentication"
	CategoryOrganization    ResourceCategory = "organization"
	CategoryTINVerification ResourceCategory = "tin_verification"
	CategoryInvoice         ResourceCategory = "invoice"
	CategoryProduct         ResourceCategory = "product"
	CategoryBilling         ResourceCategory = "billing"
)

type AuditLog struct {
	ID           uint              `json:"-" gorm:"primaryKey;unique"`
	UID          string            `gorm:"not null;unique"`
	UserID       *uint             `json:"-" gorm:"not null;index"`
	User         *User             `json:"User,omitempty" gorm:"foreignKey:UserID"`
	Description  string            `gorm:"not null"`
	Actor        string            `gorm:"not null"`
	Status       string            `gorm:"not null"`
	EventType    LogEvent          `gorm:"not null"`
	Severity     LogSeverity       `gorm:"not null"`
	Resource     string            `gorm:"not null"`
	Category     ResourceCategory  `gorm:"not null"`
	Environment  string            `gorm:"not null"`
	ResourceType string            `gorm:"not null"`
	Metadata     datatypes.JSONMap `gorm:"type:jsonb"`
	UserAgent    string            `gorm:"type:text"`
	CreatedAt    time.Time         `gorm:"autoCreateTime"`
	UpdatedAt    time.Time         `json:"-"`
}

func (a *AuditLog) BeforeCreate(tx *gorm.DB) (err error) {
	a.UID = pkg.GenerateUUIDV7()
	a.CreatedAt = time.Now().Local()
	a.UpdatedAt = time.Now().Local()

	return
}
