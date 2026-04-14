POST https://{{host}}/service/soap/AuthRequest
Content-Type: text/xml
```xml
<?xml version='1.0' encoding='UTF-8'?>
<soap:Envelope xmlns:soap="http://www.w3.org/2003/05/soap-envelope" xmlns:urn="urn:zimbra" xmlns:urn1="urn:zimbraAccount">
   <soap:Body>
      <urn1:AuthRequest >
         <urn1:password>assext</urn1:password>
         <urn1:account by="name">user@{{domain}}</urn1:account>
      </urn1:AuthRequest>
   </soap:Body>
</soap:Envelope>
```
HTTP 200
[Captures]
token: cookie "ZM_AUTH_TOKEN"

GET https://{{host}}/services/storages/quota/myself
X-Api-Version: 2
Cookie: {{token}}
HTTP 200
[Asserts]
jsonpath "$.total.computedLimit" exists
jsonpath "$.total.used" exists
jsonpath "$.modules.files.used" exists
jsonpath "$.modules.mailbox.used" exists
jsonpath "$.modules.wsc.used" exists
