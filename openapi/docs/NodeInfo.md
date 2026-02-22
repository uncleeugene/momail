# NodeInfo

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Address** | Pointer to **string** |  | [optional] 
**Sysop** | Pointer to **string** |  | [optional] 
**Location** | Pointer to **string** |  | [optional] 
**Flags** | Pointer to **[]string** |  | [optional] 
**Phone** | Pointer to **string** |  | [optional] 
**Dns** | Pointer to **string** | The resolved DNS address for the node. | [optional] 
**DnsRoot** | Pointer to **string** | The DNS root that was queried. | [optional] 

## Methods

### NewNodeInfo

`func NewNodeInfo() *NodeInfo`

NewNodeInfo instantiates a new NodeInfo object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewNodeInfoWithDefaults

`func NewNodeInfoWithDefaults() *NodeInfo`

NewNodeInfoWithDefaults instantiates a new NodeInfo object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAddress

`func (o *NodeInfo) GetAddress() string`

GetAddress returns the Address field if non-nil, zero value otherwise.

### GetAddressOk

`func (o *NodeInfo) GetAddressOk() (*string, bool)`

GetAddressOk returns a tuple with the Address field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAddress

`func (o *NodeInfo) SetAddress(v string)`

SetAddress sets Address field to given value.

### HasAddress

`func (o *NodeInfo) HasAddress() bool`

HasAddress returns a boolean if a field has been set.

### GetSysop

`func (o *NodeInfo) GetSysop() string`

GetSysop returns the Sysop field if non-nil, zero value otherwise.

### GetSysopOk

`func (o *NodeInfo) GetSysopOk() (*string, bool)`

GetSysopOk returns a tuple with the Sysop field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSysop

`func (o *NodeInfo) SetSysop(v string)`

SetSysop sets Sysop field to given value.

### HasSysop

`func (o *NodeInfo) HasSysop() bool`

HasSysop returns a boolean if a field has been set.

### GetLocation

`func (o *NodeInfo) GetLocation() string`

GetLocation returns the Location field if non-nil, zero value otherwise.

### GetLocationOk

`func (o *NodeInfo) GetLocationOk() (*string, bool)`

GetLocationOk returns a tuple with the Location field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLocation

`func (o *NodeInfo) SetLocation(v string)`

SetLocation sets Location field to given value.

### HasLocation

`func (o *NodeInfo) HasLocation() bool`

HasLocation returns a boolean if a field has been set.

### GetFlags

`func (o *NodeInfo) GetFlags() []string`

GetFlags returns the Flags field if non-nil, zero value otherwise.

### GetFlagsOk

`func (o *NodeInfo) GetFlagsOk() (*[]string, bool)`

GetFlagsOk returns a tuple with the Flags field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFlags

`func (o *NodeInfo) SetFlags(v []string)`

SetFlags sets Flags field to given value.

### HasFlags

`func (o *NodeInfo) HasFlags() bool`

HasFlags returns a boolean if a field has been set.

### GetPhone

`func (o *NodeInfo) GetPhone() string`

GetPhone returns the Phone field if non-nil, zero value otherwise.

### GetPhoneOk

`func (o *NodeInfo) GetPhoneOk() (*string, bool)`

GetPhoneOk returns a tuple with the Phone field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPhone

`func (o *NodeInfo) SetPhone(v string)`

SetPhone sets Phone field to given value.

### HasPhone

`func (o *NodeInfo) HasPhone() bool`

HasPhone returns a boolean if a field has been set.

### GetDns

`func (o *NodeInfo) GetDns() string`

GetDns returns the Dns field if non-nil, zero value otherwise.

### GetDnsOk

`func (o *NodeInfo) GetDnsOk() (*string, bool)`

GetDnsOk returns a tuple with the Dns field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDns

`func (o *NodeInfo) SetDns(v string)`

SetDns sets Dns field to given value.

### HasDns

`func (o *NodeInfo) HasDns() bool`

HasDns returns a boolean if a field has been set.

### GetDnsRoot

`func (o *NodeInfo) GetDnsRoot() string`

GetDnsRoot returns the DnsRoot field if non-nil, zero value otherwise.

### GetDnsRootOk

`func (o *NodeInfo) GetDnsRootOk() (*string, bool)`

GetDnsRootOk returns a tuple with the DnsRoot field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDnsRoot

`func (o *NodeInfo) SetDnsRoot(v string)`

SetDnsRoot sets DnsRoot field to given value.

### HasDnsRoot

`func (o *NodeInfo) HasDnsRoot() bool`

HasDnsRoot returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


