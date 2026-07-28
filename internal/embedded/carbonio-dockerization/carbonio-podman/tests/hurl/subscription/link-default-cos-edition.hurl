# Link the default COS to the "workspace" edition via the subscription extension.
#
# Flow: admin login -> resolve default COS id -> PUT edition -> verify with GET.
# The extension is gated by AdminAuthStage: it needs the ZM_ADMIN_AUTH_TOKEN
# cookie, so we forward the captured admin token on every extension call.

# Login as admin
POST https://{{host}}:6071/service/admin/soap/AuthRequest
Content-Type: text/xml
```xml
<?xml version='1.0' encoding='UTF-8'?>
<soap:Envelope xmlns:soap="http://www.w3.org/2003/05/soap-envelope" xmlns:urn="urn:zimbra" xmlns:urn1="urn:zimbraAdmin">
   <soap:Body>
      <urn1:AuthRequest>
         <urn1:password>assext</urn1:password>
         <urn1:account by="name">zextras@{{domain}}</urn1:account>
      </urn1:AuthRequest>
   </soap:Body>
</soap:Envelope>
```
HTTP 200
[Captures]
admin_token: cookie "ZM_ADMIN_AUTH_TOKEN"

# Resolve the default COS id (the extension resolves a COS by id, not name)
POST https://{{host}}:6071/service/admin/soap
Content-Type: text/xml
Cookie: ZM_ADMIN_AUTH_TOKEN={{admin_token}}
```xml
<?xml version='1.0' encoding='UTF-8'?>
<soap:Envelope xmlns:soap="http://www.w3.org/2003/05/soap-envelope" xmlns:urn="urn:zimbra" xmlns:urn1="urn:zimbraAdmin">
   <soap:Header>
      <urn:context>
         <urn:authToken>{{admin_token}}</urn:authToken>
      </urn:context>
   </soap:Header>
   <soap:Body>
      <urn1:GetCosRequest>
         <urn1:cos by="name">default</urn1:cos>
      </urn1:GetCosRequest>
   </soap:Body>
</soap:Envelope>
```
HTTP 200
[Captures]
cos_id: xpath "string(//*[local-name()='cos']/@id)"

# Link the COS to the workspace edition
PUT https://{{host}}/service/extension/subscription
Content-Type: application/json
Cookie: ZM_ADMIN_AUTH_TOKEN={{admin_token}}
```json
{
  "cosId": "{{cos_id}}",
  "edition": "workspace"
}
```
HTTP 204

# Verify the edition is now set
GET https://{{host}}/service/extension/subscription?cosId={{cos_id}}
Cookie: ZM_ADMIN_AUTH_TOKEN={{admin_token}}
HTTP 200
[Asserts]
jsonpath "$.cosId" == "{{cos_id}}"
jsonpath "$.edition" == "workspace"
jsonpath "$.totalSeats" exists
