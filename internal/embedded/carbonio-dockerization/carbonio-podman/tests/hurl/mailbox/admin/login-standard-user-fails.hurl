POST https://{{host}}:6071/service/admin/soap/AuthRequest
Content-Type: text/xml
```xml
<?xml version='1.0' encoding='UTF-8'?>
<soap:Envelope xmlns:soap="http://www.w3.org/2003/05/soap-envelope" xmlns:urn="urn:zimbra" xmlns:urn1="urn:zimbraAdmin">
   <soap:Body>
      <urn1:AuthRequest >
         <urn1:password>assext</urn1:password>
         <urn1:account by="name">user@{{domain}}</urn1:account>
      </urn1:AuthRequest>
   </soap:Body>
</soap:Envelope>
```
HTTP 422
