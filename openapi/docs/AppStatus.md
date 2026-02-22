# AppStatus

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Address** | Pointer to **string** |  | [optional] 
**Version** | Pointer to **string** |  | [optional] 
**Uptime** | Pointer to **string** |  | [optional] 
**Sessions** | Pointer to [**map[string]SessionInfo**](SessionInfo.md) |  | [optional] 
**NextScanIn** | Pointer to **int32** | Seconds until next scan | [optional] 
**Muted** | Pointer to **bool** |  | [optional] 
**OutboundQueue** | Pointer to [**[]QueueEntry**](QueueEntry.md) |  | [optional] 

## Methods

### NewAppStatus

`func NewAppStatus() *AppStatus`

NewAppStatus instantiates a new AppStatus object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewAppStatusWithDefaults

`func NewAppStatusWithDefaults() *AppStatus`

NewAppStatusWithDefaults instantiates a new AppStatus object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAddress

`func (o *AppStatus) GetAddress() string`

GetAddress returns the Address field if non-nil, zero value otherwise.

### GetAddressOk

`func (o *AppStatus) GetAddressOk() (*string, bool)`

GetAddressOk returns a tuple with the Address field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAddress

`func (o *AppStatus) SetAddress(v string)`

SetAddress sets Address field to given value.

### HasAddress

`func (o *AppStatus) HasAddress() bool`

HasAddress returns a boolean if a field has been set.

### GetVersion

`func (o *AppStatus) GetVersion() string`

GetVersion returns the Version field if non-nil, zero value otherwise.

### GetVersionOk

`func (o *AppStatus) GetVersionOk() (*string, bool)`

GetVersionOk returns a tuple with the Version field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetVersion

`func (o *AppStatus) SetVersion(v string)`

SetVersion sets Version field to given value.

### HasVersion

`func (o *AppStatus) HasVersion() bool`

HasVersion returns a boolean if a field has been set.

### GetUptime

`func (o *AppStatus) GetUptime() string`

GetUptime returns the Uptime field if non-nil, zero value otherwise.

### GetUptimeOk

`func (o *AppStatus) GetUptimeOk() (*string, bool)`

GetUptimeOk returns a tuple with the Uptime field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUptime

`func (o *AppStatus) SetUptime(v string)`

SetUptime sets Uptime field to given value.

### HasUptime

`func (o *AppStatus) HasUptime() bool`

HasUptime returns a boolean if a field has been set.

### GetSessions

`func (o *AppStatus) GetSessions() map[string]SessionInfo`

GetSessions returns the Sessions field if non-nil, zero value otherwise.

### GetSessionsOk

`func (o *AppStatus) GetSessionsOk() (*map[string]SessionInfo, bool)`

GetSessionsOk returns a tuple with the Sessions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSessions

`func (o *AppStatus) SetSessions(v map[string]SessionInfo)`

SetSessions sets Sessions field to given value.

### HasSessions

`func (o *AppStatus) HasSessions() bool`

HasSessions returns a boolean if a field has been set.

### GetNextScanIn

`func (o *AppStatus) GetNextScanIn() int32`

GetNextScanIn returns the NextScanIn field if non-nil, zero value otherwise.

### GetNextScanInOk

`func (o *AppStatus) GetNextScanInOk() (*int32, bool)`

GetNextScanInOk returns a tuple with the NextScanIn field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNextScanIn

`func (o *AppStatus) SetNextScanIn(v int32)`

SetNextScanIn sets NextScanIn field to given value.

### HasNextScanIn

`func (o *AppStatus) HasNextScanIn() bool`

HasNextScanIn returns a boolean if a field has been set.

### GetMuted

`func (o *AppStatus) GetMuted() bool`

GetMuted returns the Muted field if non-nil, zero value otherwise.

### GetMutedOk

`func (o *AppStatus) GetMutedOk() (*bool, bool)`

GetMutedOk returns a tuple with the Muted field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMuted

`func (o *AppStatus) SetMuted(v bool)`

SetMuted sets Muted field to given value.

### HasMuted

`func (o *AppStatus) HasMuted() bool`

HasMuted returns a boolean if a field has been set.

### GetOutboundQueue

`func (o *AppStatus) GetOutboundQueue() []QueueEntry`

GetOutboundQueue returns the OutboundQueue field if non-nil, zero value otherwise.

### GetOutboundQueueOk

`func (o *AppStatus) GetOutboundQueueOk() (*[]QueueEntry, bool)`

GetOutboundQueueOk returns a tuple with the OutboundQueue field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOutboundQueue

`func (o *AppStatus) SetOutboundQueue(v []QueueEntry)`

SetOutboundQueue sets OutboundQueue field to given value.

### HasOutboundQueue

`func (o *AppStatus) HasOutboundQueue() bool`

HasOutboundQueue returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


