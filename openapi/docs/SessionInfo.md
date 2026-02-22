# SessionInfo

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | Pointer to **string** |  | [optional] 
**Remote** | Pointer to **string** |  | [optional] 
**Direction** | Pointer to **string** |  | [optional] 
**State** | Pointer to **string** |  | [optional] 
**CurrentFile** | Pointer to **string** |  | [optional] 
**FilePos** | Pointer to **int64** |  | [optional] 
**FileSize** | Pointer to **int64** |  | [optional] 
**StartedAt** | Pointer to **time.Time** |  | [optional] 

## Methods

### NewSessionInfo

`func NewSessionInfo() *SessionInfo`

NewSessionInfo instantiates a new SessionInfo object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewSessionInfoWithDefaults

`func NewSessionInfoWithDefaults() *SessionInfo`

NewSessionInfoWithDefaults instantiates a new SessionInfo object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetId

`func (o *SessionInfo) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *SessionInfo) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *SessionInfo) SetId(v string)`

SetId sets Id field to given value.

### HasId

`func (o *SessionInfo) HasId() bool`

HasId returns a boolean if a field has been set.

### GetRemote

`func (o *SessionInfo) GetRemote() string`

GetRemote returns the Remote field if non-nil, zero value otherwise.

### GetRemoteOk

`func (o *SessionInfo) GetRemoteOk() (*string, bool)`

GetRemoteOk returns a tuple with the Remote field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRemote

`func (o *SessionInfo) SetRemote(v string)`

SetRemote sets Remote field to given value.

### HasRemote

`func (o *SessionInfo) HasRemote() bool`

HasRemote returns a boolean if a field has been set.

### GetDirection

`func (o *SessionInfo) GetDirection() string`

GetDirection returns the Direction field if non-nil, zero value otherwise.

### GetDirectionOk

`func (o *SessionInfo) GetDirectionOk() (*string, bool)`

GetDirectionOk returns a tuple with the Direction field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDirection

`func (o *SessionInfo) SetDirection(v string)`

SetDirection sets Direction field to given value.

### HasDirection

`func (o *SessionInfo) HasDirection() bool`

HasDirection returns a boolean if a field has been set.

### GetState

`func (o *SessionInfo) GetState() string`

GetState returns the State field if non-nil, zero value otherwise.

### GetStateOk

`func (o *SessionInfo) GetStateOk() (*string, bool)`

GetStateOk returns a tuple with the State field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetState

`func (o *SessionInfo) SetState(v string)`

SetState sets State field to given value.

### HasState

`func (o *SessionInfo) HasState() bool`

HasState returns a boolean if a field has been set.

### GetCurrentFile

`func (o *SessionInfo) GetCurrentFile() string`

GetCurrentFile returns the CurrentFile field if non-nil, zero value otherwise.

### GetCurrentFileOk

`func (o *SessionInfo) GetCurrentFileOk() (*string, bool)`

GetCurrentFileOk returns a tuple with the CurrentFile field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCurrentFile

`func (o *SessionInfo) SetCurrentFile(v string)`

SetCurrentFile sets CurrentFile field to given value.

### HasCurrentFile

`func (o *SessionInfo) HasCurrentFile() bool`

HasCurrentFile returns a boolean if a field has been set.

### GetFilePos

`func (o *SessionInfo) GetFilePos() int64`

GetFilePos returns the FilePos field if non-nil, zero value otherwise.

### GetFilePosOk

`func (o *SessionInfo) GetFilePosOk() (*int64, bool)`

GetFilePosOk returns a tuple with the FilePos field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFilePos

`func (o *SessionInfo) SetFilePos(v int64)`

SetFilePos sets FilePos field to given value.

### HasFilePos

`func (o *SessionInfo) HasFilePos() bool`

HasFilePos returns a boolean if a field has been set.

### GetFileSize

`func (o *SessionInfo) GetFileSize() int64`

GetFileSize returns the FileSize field if non-nil, zero value otherwise.

### GetFileSizeOk

`func (o *SessionInfo) GetFileSizeOk() (*int64, bool)`

GetFileSizeOk returns a tuple with the FileSize field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFileSize

`func (o *SessionInfo) SetFileSize(v int64)`

SetFileSize sets FileSize field to given value.

### HasFileSize

`func (o *SessionInfo) HasFileSize() bool`

HasFileSize returns a boolean if a field has been set.

### GetStartedAt

`func (o *SessionInfo) GetStartedAt() time.Time`

GetStartedAt returns the StartedAt field if non-nil, zero value otherwise.

### GetStartedAtOk

`func (o *SessionInfo) GetStartedAtOk() (*time.Time, bool)`

GetStartedAtOk returns a tuple with the StartedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStartedAt

`func (o *SessionInfo) SetStartedAt(v time.Time)`

SetStartedAt sets StartedAt field to given value.

### HasStartedAt

`func (o *SessionInfo) HasStartedAt() bool`

HasStartedAt returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


