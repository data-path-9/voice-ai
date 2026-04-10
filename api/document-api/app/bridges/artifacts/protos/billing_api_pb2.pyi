import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
import app.bridges.artifacts.protos.common_pb2 as _common_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class BillingPlanQuota(_message.Message):
    __slots__ = ("id", "resourceType", "quotaLimit")
    ID_FIELD_NUMBER: _ClassVar[int]
    RESOURCETYPE_FIELD_NUMBER: _ClassVar[int]
    QUOTALIMIT_FIELD_NUMBER: _ClassVar[int]
    id: int
    resourceType: str
    quotaLimit: int
    def __init__(self, id: _Optional[int] = ..., resourceType: _Optional[str] = ..., quotaLimit: _Optional[int] = ...) -> None: ...

class BillingPlan(_message.Message):
    __slots__ = ("id", "name", "slug", "description", "isDefault", "isActive", "sortOrder", "priceMonthly", "priceYearly", "currency", "stripeUrl", "quotas")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    SLUG_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    ISDEFAULT_FIELD_NUMBER: _ClassVar[int]
    ISACTIVE_FIELD_NUMBER: _ClassVar[int]
    SORTORDER_FIELD_NUMBER: _ClassVar[int]
    PRICEMONTHLY_FIELD_NUMBER: _ClassVar[int]
    PRICEYEARLY_FIELD_NUMBER: _ClassVar[int]
    CURRENCY_FIELD_NUMBER: _ClassVar[int]
    STRIPEURL_FIELD_NUMBER: _ClassVar[int]
    QUOTAS_FIELD_NUMBER: _ClassVar[int]
    id: int
    name: str
    slug: str
    description: str
    isDefault: bool
    isActive: bool
    sortOrder: int
    priceMonthly: int
    priceYearly: int
    currency: str
    stripeUrl: str
    quotas: _containers.RepeatedCompositeFieldContainer[BillingPlanQuota]
    def __init__(self, id: _Optional[int] = ..., name: _Optional[str] = ..., slug: _Optional[str] = ..., description: _Optional[str] = ..., isDefault: bool = ..., isActive: bool = ..., sortOrder: _Optional[int] = ..., priceMonthly: _Optional[int] = ..., priceYearly: _Optional[int] = ..., currency: _Optional[str] = ..., stripeUrl: _Optional[str] = ..., quotas: _Optional[_Iterable[_Union[BillingPlanQuota, _Mapping]]] = ...) -> None: ...

class BillingSubscription(_message.Message):
    __slots__ = ("id", "organizationId", "billingPlanId", "status", "billingInterval", "currentPeriodStart", "currentPeriodEnd", "plan", "createdDate", "updatedDate")
    ID_FIELD_NUMBER: _ClassVar[int]
    ORGANIZATIONID_FIELD_NUMBER: _ClassVar[int]
    BILLINGPLANID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    BILLINGINTERVAL_FIELD_NUMBER: _ClassVar[int]
    CURRENTPERIODSTART_FIELD_NUMBER: _ClassVar[int]
    CURRENTPERIODEND_FIELD_NUMBER: _ClassVar[int]
    PLAN_FIELD_NUMBER: _ClassVar[int]
    CREATEDDATE_FIELD_NUMBER: _ClassVar[int]
    UPDATEDDATE_FIELD_NUMBER: _ClassVar[int]
    id: int
    organizationId: int
    billingPlanId: int
    status: str
    billingInterval: str
    currentPeriodStart: _timestamp_pb2.Timestamp
    currentPeriodEnd: _timestamp_pb2.Timestamp
    plan: BillingPlan
    createdDate: _timestamp_pb2.Timestamp
    updatedDate: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[int] = ..., organizationId: _Optional[int] = ..., billingPlanId: _Optional[int] = ..., status: _Optional[str] = ..., billingInterval: _Optional[str] = ..., currentPeriodStart: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., currentPeriodEnd: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., plan: _Optional[_Union[BillingPlan, _Mapping]] = ..., createdDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updatedDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class GetAllPlansRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetAllPlansResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: _containers.RepeatedCompositeFieldContainer[BillingPlan]
    error: _common_pb2.Error
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Iterable[_Union[BillingPlan, _Mapping]]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ...) -> None: ...

class GetSubscriptionRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetSubscriptionResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: BillingSubscription
    error: _common_pb2.Error
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Union[BillingSubscription, _Mapping]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ...) -> None: ...

class UpdateSubscriptionRequest(_message.Message):
    __slots__ = ("planSlug",)
    PLANSLUG_FIELD_NUMBER: _ClassVar[int]
    planSlug: str
    def __init__(self, planSlug: _Optional[str] = ...) -> None: ...

class UpdateSubscriptionResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: BillingSubscription
    error: _common_pb2.Error
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Union[BillingSubscription, _Mapping]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ...) -> None: ...
