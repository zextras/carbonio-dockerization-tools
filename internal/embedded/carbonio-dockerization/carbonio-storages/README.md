This is a server that mocks storages while we don't have the image available in the registry.

I could have built the image like docs editor using an ubuntu base image and installing with apt
but since the endpoints were fairly simple I just chose to mock it: it's faster and it gets the job done.