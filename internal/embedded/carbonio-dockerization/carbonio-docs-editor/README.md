The Dockerfile here is useful to build the image of docs editor locally since
it's not that easy to translate to CI.

Anyway since releases with important changes are rare, for now I built&pushed 
images manually so one can still use them precompiled without having to deal with the build each time.

Still, since this Dockerfile just installs docs editor from apt, I'm not going to move it in the
docs editor repository: apart from not using the repository itself, this Dockerfile is used only in this project for now.