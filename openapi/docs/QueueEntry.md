# QueueEntry

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Address** | Pointer to **string** |  | [optional] 
**Flavor** | Pointer to **string** | The flavor of the mail in queue (priority). | [optional] 
**Files** | Pointer to **int32** | Number of mail files in the queue. | [optional] 
**NetmailSize** | Pointer to **int64** | Total size of netmail bundles in bytes. | [optional] 
**EchomailSize** | Pointer to **int64** | Total size of echomail bundles in bytes. | [optional] 
**IsSuspended** | Pointer to **bool** | Whether the node is suspended (on hold). | [optional] 
**IsBusy** | Pointer to **bool** | Whether the node is currently busy/locked. | [optional] 

## Methods

### NewQueueEntry

`func NewQueueEntry() *QueueEntry`

NewQueueEntry instantiates a new QueueEntry object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewQueueEntryWithDefaults

`func NewQueueEntryWithDefaults() *QueueEntry`

NewQueueEntryWithDefaults instantiates a new QueueEntry object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAddress

`func (o *QueueEntry) GetAddress() string`

GetAddress returns the Address field if non-nil, zero value otherwise.

### GetAddressOk

`func (o *QueueEntry) GetAddressOk() (*string, bool)`

GetAddressOk returns a tuple with the Address field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAddress

`func (o *QueueEntry) SetAddress(v string)`

SetAddress sets Address field to given value.

### HasAddress

`func (o *QueueEntry) HasAddress() bool`

HasAddress returns a boolean if a field has been set.

### GetFlavor

`func (o *QueueEntry) GetFlavor() string`

GetFlavor returns the Flavor field if non-nil, zero value otherwise.

### GetFlavorOk

`func (o *QueueEntry) GetFlavorOk() (*string, bool)`

GetFlavorOk returns a tuple with the Flavor field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFlavor

`func (o *QueueEntry) SetFlavor(v string)`

SetFlavor sets Flavor field to given value.

### HasFlavor

`func (o *QueueEntry) HasFlavor() bool`

HasFlavor returns a boolean if a field has been set.

### GetFiles

`func (o *QueueEntry) GetFiles() int32`

GetFiles returns the Files field if non-nil, zero value otherwise.

### GetFilesOk

`func (o *QueueEntry) GetFilesOk() (*int32, bool)`

GetFilesOk returns a tuple with the Files field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFiles

`func (o *QueueEntry) SetFiles(v int32)`

SetFiles sets Files field to given value.

### HasFiles

`func (o *QueueEntry) HasFiles() bool`

HasFiles returns a boolean if a field has been set.

### GetNetmailSize

`func (o *QueueEntry) GetNetmailSize() int64`

GetNetmailSize returns the NetmailSize field if non-nil, zero value otherwise.

### GetNetmailSizeOk

`func (o *QueueEntry) GetNetmailSizeOk() (*int64, bool)`

GetNetmailSizeOk returns a tuple with the NetmailSize field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNetmailSize

`func (o *QueueEntry) SetNetmailSize(v int64)`

SetNetmailSize sets NetmailSize field to given value.

### HasNetmailSize

`func (o *QueueEntry) HasNetmailSize() bool`

HasNetmailSize returns a boolean if a field has been set.

### GetEchomailSize

`func (o *QueueEntry) GetEchomailSize() int64`

GetEchomailSize returns the EchomailSize field if non-nil, zero value otherwise.

### GetEchomailSizeOk

`func (o *QueueEntry) GetEchomailSizeOk() (*int64, bool)`

GetEchomailSizeOk returns a tuple with the EchomailSize field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEchomailSize

`func (o *QueueEntry) SetEchomailSize(v int64)`

SetEchomailSize sets EchomailSize field to given value.

### HasEchomailSize

`func (o *QueueEntry) HasEchomailSize() bool`

HasEchomailSize returns a boolean if a field has been set.

### GetIsSuspended

`func (o *QueueEntry) GetIsSuspended() bool`

GetIsSuspended returns the IsSuspended field if non-nil, zero value otherwise.

### GetIsSuspendedOk

`func (o *QueueEntry) GetIsSuspendedOk() (*bool, bool)`

GetIsSuspendedOk returns a tuple with the IsSuspended field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIsSuspended

`func (o *QueueEntry) SetIsSuspended(v bool)`

SetIsSuspended sets IsSuspended field to given value.

### HasIsSuspended

`func (o *QueueEntry) HasIsSuspended() bool`

HasIsSuspended returns a boolean if a field has been set.

### GetIsBusy

`func (o *QueueEntry) GetIsBusy() bool`

GetIsBusy returns the IsBusy field if non-nil, zero value otherwise.

### GetIsBusyOk

`func (o *QueueEntry) GetIsBusyOk() (*bool, bool)`

GetIsBusyOk returns a tuple with the IsBusy field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIsBusy

`func (o *QueueEntry) SetIsBusy(v bool)`

SetIsBusy sets IsBusy field to given value.

### HasIsBusy

`func (o *QueueEntry) HasIsBusy() bool`

HasIsBusy returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


