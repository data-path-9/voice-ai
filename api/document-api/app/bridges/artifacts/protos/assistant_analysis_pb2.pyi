import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
import app.bridges.artifacts.protos.common_pb2 as _common_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class AssistantAnalysis(_message.Message):
    __slots__ = ("id", "name", "description", "provider", "options", "assistantId", "status", "createdBy", "createdUser", "updatedBy", "updatedUser", "createdDate", "updatedDate", "executionPriority")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    OPTIONS_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CREATEDBY_FIELD_NUMBER: _ClassVar[int]
    CREATEDUSER_FIELD_NUMBER: _ClassVar[int]
    UPDATEDBY_FIELD_NUMBER: _ClassVar[int]
    UPDATEDUSER_FIELD_NUMBER: _ClassVar[int]
    CREATEDDATE_FIELD_NUMBER: _ClassVar[int]
    UPDATEDDATE_FIELD_NUMBER: _ClassVar[int]
    EXECUTIONPRIORITY_FIELD_NUMBER: _ClassVar[int]
    id: int
    name: str
    description: str
    provider: str
    options: _containers.RepeatedCompositeFieldContainer[_common_pb2.Metadata]
    assistantId: int
    status: str
    createdBy: int
    createdUser: _common_pb2.User
    updatedBy: int
    updatedUser: _common_pb2.User
    createdDate: _timestamp_pb2.Timestamp
    updatedDate: _timestamp_pb2.Timestamp
    executionPriority: int
    def __init__(self, id: _Optional[int] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., provider: _Optional[str] = ..., options: _Optional[_Iterable[_Union[_common_pb2.Metadata, _Mapping]]] = ..., assistantId: _Optional[int] = ..., status: _Optional[str] = ..., createdBy: _Optional[int] = ..., createdUser: _Optional[_Union[_common_pb2.User, _Mapping]] = ..., updatedBy: _Optional[int] = ..., updatedUser: _Optional[_Union[_common_pb2.User, _Mapping]] = ..., createdDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updatedDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., executionPriority: _Optional[int] = ...) -> None: ...

class CreateAssistantAnalysisRequest(_message.Message):
    __slots__ = ("name", "description", "provider", "options", "assistantId", "executionPriority")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    OPTIONS_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTID_FIELD_NUMBER: _ClassVar[int]
    EXECUTIONPRIORITY_FIELD_NUMBER: _ClassVar[int]
    name: str
    description: str
    provider: str
    options: _containers.RepeatedCompositeFieldContainer[_common_pb2.Metadata]
    assistantId: int
    executionPriority: int
    def __init__(self, name: _Optional[str] = ..., description: _Optional[str] = ..., provider: _Optional[str] = ..., options: _Optional[_Iterable[_Union[_common_pb2.Metadata, _Mapping]]] = ..., assistantId: _Optional[int] = ..., executionPriority: _Optional[int] = ...) -> None: ...

class UpdateAssistantAnalysisRequest(_message.Message):
    __slots__ = ("id", "name", "description", "provider", "options", "assistantId", "executionPriority")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    OPTIONS_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTID_FIELD_NUMBER: _ClassVar[int]
    EXECUTIONPRIORITY_FIELD_NUMBER: _ClassVar[int]
    id: int
    name: str
    description: str
    provider: str
    options: _containers.RepeatedCompositeFieldContainer[_common_pb2.Metadata]
    assistantId: int
    executionPriority: int
    def __init__(self, id: _Optional[int] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., provider: _Optional[str] = ..., options: _Optional[_Iterable[_Union[_common_pb2.Metadata, _Mapping]]] = ..., assistantId: _Optional[int] = ..., executionPriority: _Optional[int] = ...) -> None: ...

class GetAssistantAnalysisRequest(_message.Message):
    __slots__ = ("id", "assistantId")
    ID_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTID_FIELD_NUMBER: _ClassVar[int]
    id: int
    assistantId: int
    def __init__(self, id: _Optional[int] = ..., assistantId: _Optional[int] = ...) -> None: ...

class DeleteAssistantAnalysisRequest(_message.Message):
    __slots__ = ("id", "assistantId")
    ID_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTID_FIELD_NUMBER: _ClassVar[int]
    id: int
    assistantId: int
    def __init__(self, id: _Optional[int] = ..., assistantId: _Optional[int] = ...) -> None: ...

class GetAssistantAnalysisResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: AssistantAnalysis
    error: _common_pb2.Error
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Union[AssistantAnalysis, _Mapping]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ...) -> None: ...

class GetAllAssistantAnalysisRequest(_message.Message):
    __slots__ = ("assistantId", "paginate", "criterias")
    ASSISTANTID_FIELD_NUMBER: _ClassVar[int]
    PAGINATE_FIELD_NUMBER: _ClassVar[int]
    CRITERIAS_FIELD_NUMBER: _ClassVar[int]
    assistantId: int
    paginate: _common_pb2.Paginate
    criterias: _containers.RepeatedCompositeFieldContainer[_common_pb2.Criteria]
    def __init__(self, assistantId: _Optional[int] = ..., paginate: _Optional[_Union[_common_pb2.Paginate, _Mapping]] = ..., criterias: _Optional[_Iterable[_Union[_common_pb2.Criteria, _Mapping]]] = ...) -> None: ...

class GetAllAssistantAnalysisResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error", "paginated")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    PAGINATED_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: _containers.RepeatedCompositeFieldContainer[AssistantAnalysis]
    error: _common_pb2.Error
    paginated: _common_pb2.Paginated
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Iterable[_Union[AssistantAnalysis, _Mapping]]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ..., paginated: _Optional[_Union[_common_pb2.Paginated, _Mapping]] = ...) -> None: ...
