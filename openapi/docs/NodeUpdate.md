# NodeUpdate

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Flavor** | Pointer to **string** |  | [optional] 
**Suspended** | Pointer to **bool** |  | [optional] 

## Methods

### NewNodeUpdate

`func NewNodeUpdate() *NodeUpdate`

NewNodeUpdate instantiates a new NodeUpdate object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewNodeUpdateWithDefaults

`func NewNodeUpdateWithDefaults() *NodeUpdate`

NewNodeUpdateWithDefaults instantiates a new NodeUpdate object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetFlavor

`func (o *NodeUpdate) GetFlavor() string`

GetFlavor returns the Flavor field if non-nil, zero value otherwise.

### GetFlavorOk

`func (o *NodeUpdate) GetFlavorOk() (*string, bool)`

GetFlavorOk returns a tuple with the Flavor field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFlavor

`func (o *NodeUpdate) SetFlavor(v string)`

SetFlavor sets Flavor field to given value.

### HasFlavor

`func (o *NodeUpdate) HasFlavor() bool`

HasFlavor returns a boolean if a field has been set.

### GetSuspended

`func (o *NodeUpdate) GetSuspended() bool`

GetSuspended returns the Suspended field if non-nil, zero value otherwise.

### GetSuspendedOk

`func (o *NodeUpdate) GetSuspendedOk() (*bool, bool)`

GetSuspendedOk returns a tuple with the Suspended field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSuspended

`func (o *NodeUpdate) SetSuspended(v bool)`

SetSuspended sets Suspended field to given value.

### HasSuspended

`func (o *NodeUpdate) HasSuspended() bool`

HasSuspended returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


