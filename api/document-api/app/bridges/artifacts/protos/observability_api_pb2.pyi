import datetime

import app.bridges.artifacts.protos.common_pb2 as _common_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ObservabilityRecordKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    OBSERVABILITY_RECORD_KIND_UNSPECIFIED: _ClassVar[ObservabilityRecordKind]
    OBSERVABILITY_RECORD_KIND_LOG: _ClassVar[ObservabilityRecordKind]
    OBSERVABILITY_RECORD_KIND_EVENT: _ClassVar[ObservabilityRecordKind]
    OBSERVABILITY_RECORD_KIND_METRIC: _ClassVar[ObservabilityRecordKind]
OBSERVABILITY_RECORD_KIND_UNSPECIFIED: ObservabilityRecordKind
OBSERVABILITY_RECORD_KIND_LOG: ObservabilityRecordKind
OBSERVABILITY_RECORD_KIND_EVENT: ObservabilityRecordKind
OBSERVABILITY_RECORD_KIND_METRIC: ObservabilityRecordKind

class GetAllTelemetryRequest(_message.Message):
    __slots__ = ("paginate", "criterias", "order")
    PAGINATE_FIELD_NUMBER: _ClassVar[int]
    CRITERIAS_FIELD_NUMBER: _ClassVar[int]
    ORDER_FIELD_NUMBER: _ClassVar[int]
    paginate: _common_pb2.Paginate
    criterias: _containers.RepeatedCompositeFieldContainer[_common_pb2.Criteria]
    order: _common_pb2.Ordering
    def __init__(self, paginate: _Optional[_Union[_common_pb2.Paginate, _Mapping]] = ..., criterias: _Optional[_Iterable[_Union[_common_pb2.Criteria, _Mapping]]] = ..., order: _Optional[_Union[_common_pb2.Ordering, _Mapping]] = ...) -> None: ...

class ObservabilityLogRecord(_message.Message):
    __slots__ = ("id", "kind", "level", "message", "projectId", "organizationId", "scope", "scopeAttributes", "attributes", "occurredAt")
    class ScopeAttributesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    class AttributesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    LEVEL_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    PROJECTID_FIELD_NUMBER: _ClassVar[int]
    ORGANIZATIONID_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    SCOPEATTRIBUTES_FIELD_NUMBER: _ClassVar[int]
    ATTRIBUTES_FIELD_NUMBER: _ClassVar[int]
    OCCURREDAT_FIELD_NUMBER: _ClassVar[int]
    id: str
    kind: ObservabilityRecordKind
    level: str
    message: str
    projectId: int
    organizationId: int
    scope: str
    scopeAttributes: _containers.ScalarMap[str, str]
    attributes: _containers.ScalarMap[str, str]
    occurredAt: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., kind: _Optional[_Union[ObservabilityRecordKind, str]] = ..., level: _Optional[str] = ..., message: _Optional[str] = ..., projectId: _Optional[int] = ..., organizationId: _Optional[int] = ..., scope: _Optional[str] = ..., scopeAttributes: _Optional[_Mapping[str, str]] = ..., attributes: _Optional[_Mapping[str, str]] = ..., occurredAt: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ObservabilityEventRecord(_message.Message):
    __slots__ = ("id", "kind", "event", "component", "projectId", "organizationId", "scope", "scopeAttributes", "attributes", "occurredAt")
    class ScopeAttributesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    class AttributesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    EVENT_FIELD_NUMBER: _ClassVar[int]
    COMPONENT_FIELD_NUMBER: _ClassVar[int]
    PROJECTID_FIELD_NUMBER: _ClassVar[int]
    ORGANIZATIONID_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    SCOPEATTRIBUTES_FIELD_NUMBER: _ClassVar[int]
    ATTRIBUTES_FIELD_NUMBER: _ClassVar[int]
    OCCURREDAT_FIELD_NUMBER: _ClassVar[int]
    id: str
    kind: ObservabilityRecordKind
    event: str
    component: str
    projectId: int
    organizationId: int
    scope: str
    scopeAttributes: _containers.ScalarMap[str, str]
    attributes: _containers.ScalarMap[str, str]
    occurredAt: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., kind: _Optional[_Union[ObservabilityRecordKind, str]] = ..., event: _Optional[str] = ..., component: _Optional[str] = ..., projectId: _Optional[int] = ..., organizationId: _Optional[int] = ..., scope: _Optional[str] = ..., scopeAttributes: _Optional[_Mapping[str, str]] = ..., attributes: _Optional[_Mapping[str, str]] = ..., occurredAt: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ObservabilityMetricRecord(_message.Message):
    __slots__ = ("id", "kind", "name", "value", "description", "projectId", "organizationId", "scope", "scopeAttributes", "attributes", "occurredAt")
    class ScopeAttributesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    class AttributesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    PROJECTID_FIELD_NUMBER: _ClassVar[int]
    ORGANIZATIONID_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    SCOPEATTRIBUTES_FIELD_NUMBER: _ClassVar[int]
    ATTRIBUTES_FIELD_NUMBER: _ClassVar[int]
    OCCURREDAT_FIELD_NUMBER: _ClassVar[int]
    id: str
    kind: ObservabilityRecordKind
    name: str
    value: str
    description: str
    projectId: int
    organizationId: int
    scope: str
    scopeAttributes: _containers.ScalarMap[str, str]
    attributes: _containers.ScalarMap[str, str]
    occurredAt: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., kind: _Optional[_Union[ObservabilityRecordKind, str]] = ..., name: _Optional[str] = ..., value: _Optional[str] = ..., description: _Optional[str] = ..., projectId: _Optional[int] = ..., organizationId: _Optional[int] = ..., scope: _Optional[str] = ..., scopeAttributes: _Optional[_Mapping[str, str]] = ..., attributes: _Optional[_Mapping[str, str]] = ..., occurredAt: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ObservabilityRecord(_message.Message):
    __slots__ = ("log", "event", "metric")
    LOG_FIELD_NUMBER: _ClassVar[int]
    EVENT_FIELD_NUMBER: _ClassVar[int]
    METRIC_FIELD_NUMBER: _ClassVar[int]
    log: ObservabilityLogRecord
    event: ObservabilityEventRecord
    metric: ObservabilityMetricRecord
    def __init__(self, log: _Optional[_Union[ObservabilityLogRecord, _Mapping]] = ..., event: _Optional[_Union[ObservabilityEventRecord, _Mapping]] = ..., metric: _Optional[_Union[ObservabilityMetricRecord, _Mapping]] = ...) -> None: ...

class GetAllTelemetryResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error", "paginated")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    PAGINATED_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: _containers.RepeatedCompositeFieldContainer[ObservabilityRecord]
    error: _common_pb2.Error
    paginated: _common_pb2.Paginated
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Iterable[_Union[ObservabilityRecord, _Mapping]]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ..., paginated: _Optional[_Union[_common_pb2.Paginated, _Mapping]] = ...) -> None: ...
