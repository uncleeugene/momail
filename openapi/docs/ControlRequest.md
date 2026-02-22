# ControlRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Action** | **string** | The control action to perform. | 
**Address** | Pointer to **string** | The FidoNet address for &#39;queue_poll&#39; action. | [optional] 
**Flavor** | Pointer to **string** | The flavor of poll file to create. Can be &#39;crash&#39;, &#39;direct&#39;, &#39;hold&#39;, or &#39;normal&#39;. | [optional] [default to "normal"]
**Command** | Pointer to **string** | The exact command string for &#39;run_task&#39; action, as defined in config. | [optional] 

## Methods

### NewControlRequest

`func NewControlRequest(action string, ) *ControlRequest`

NewControlRequest instantiates a new ControlRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewControlRequestWithDefaults

`func NewControlRequestWithDefaults() *ControlRequest`

NewControlRequestWithDefaults instantiates a new ControlRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAction

`func (o *ControlRequest) GetAction() string`

GetAction returns the Action field if non-nil, zero value otherwise.

### GetActionOk

`func (o *ControlRequest) GetActionOk() (*string, bool)`

GetActionOk returns a tuple with the Action field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAction

`func (o *ControlRequest) SetAction(v string)`

SetAction sets Action field to given value.


### GetAddress

`func (o *ControlRequest) GetAddress() string`

GetAddress returns the Address field if non-nil, zero value otherwise.

### GetAddressOk

`func (o *ControlRequest) GetAddressOk() (*string, bool)`

GetAddressOk returns a tuple with the Address field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAddress

`func (o *ControlRequest) SetAddress(v string)`

SetAddress sets Address field to given value.

### HasAddress

`func (o *ControlRequest) HasAddress() bool`

HasAddress returns a boolean if a field has been set.

### GetFlavor

`func (o *ControlRequest) GetFlavor() string`

GetFlavor returns the Flavor field if non-nil, zero value otherwise.

### GetFlavorOk

`func (o *ControlRequest) GetFlavorOk() (*string, bool)`

GetFlavorOk returns a tuple with the Flavor field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFlavor

`func (o *ControlRequest) SetFlavor(v string)`

SetFlavor sets Flavor field to given value.

### HasFlavor

`func (o *ControlRequest) HasFlavor() bool`

HasFlavor returns a boolean if a field has been set.

### GetCommand

`func (o *ControlRequest) GetCommand() string`

GetCommand returns the Command field if non-nil, zero value otherwise.

### GetCommandOk

`func (o *ControlRequest) GetCommandOk() (*string, bool)`

GetCommandOk returns a tuple with the Command field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCommand

`func (o *ControlRequest) SetCommand(v string)`

SetCommand sets Command field to given value.

### HasCommand

`func (o *ControlRequest) HasCommand() bool`

HasCommand returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


