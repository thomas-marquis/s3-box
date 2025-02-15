# s3-box
A simple desktop application for accessing s3 ressources


## DDD

* Connection (bucket, creds, settings)
    * States: Created, up, down, selected, not selected, deleted
    * Rules : Une conection down ne peut pas être selectionnée
* bucket (rootdir, name, connection)
    * State: Created, opened, closed
* S3 directory
    * n'existe que si contient au moins un fichier => le seul moyen de créer un sous dossier est d'y créer un fichier
    * peut contenur d'autres s3 directories
    * States : created, opened, closed, deleted
    * Action: créer ou modifier un fichier dans le dossier
    * Action: créer un fichier dans un nouveau sous-dossier
* S3 file
    * est contenu dans un et un seul directory
    * states: created, deleted
* local file


