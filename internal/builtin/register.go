package builtin

import "lunex/internal/compiler"

func RegisterAll(c *compiler.Compiler) {
        interp := c.Interpreter()
        runtimeMod := RuntimeModule(interp)
        interp.RegisterModule("runtime", runtimeMod)

        ioMod := IoModule()
        fsMod := FsModule()
        httpMod := HttpModule()
        cryptoMod := CryptoModule()
        dbMod := DbModule()
        envMod := EnvModule()
        wsMod := WsModule()
        mailMod := MailModule()
        aiMod := AiModule()
        utilsMod := UtilsModule()
        validateMod := ValidateModule()
        postgresMod := PostgresModule()
        redisMod := RedisModule()
        jwtMod := JWTModule()
        mysqlMod := MySQLModule()
        stripeMod := StripeModule()
        oauth2Mod := OAuth2Module()
        graphqlMod := GraphQLModule()
        rabbitmqMod := RabbitMQModule()
        excelMod := ExcelModule()
        pdfMod := PDFModule()
        csvMod := CSVModule()
        yamlMod := YAMLModule()
        tomlMod := TOMLModule()
        markdownMod := MarkdownModule()
        mustacheMod := MustacheModule()
        xmlMod := XMLModule()
        osMod := OsModule()
        allocMod := AllocModule()
        zipMod := ZipModule()
        regexMod := RegexModule()
        mathMod := MathModule()
        datetimeMod := DatetimeModule()
        pathMod := PathModule()
        compressMod := CompressModule()

        native := NativeModule()
        native.ObjVal["io"] = ioMod
        native.ObjVal["fs"] = fsMod
        native.ObjVal["http"] = httpMod
        native.ObjVal["crypto"] = cryptoMod
        native.ObjVal["db"] = dbMod
        native.ObjVal["env"] = envMod
        native.ObjVal["ws"] = wsMod
        native.ObjVal["mail"] = mailMod
        native.ObjVal["ai"] = aiMod
        native.ObjVal["utils"] = utilsMod
        native.ObjVal["validate"] = validateMod
        native.ObjVal["postgres"] = postgresMod
        native.ObjVal["redis"] = redisMod
        native.ObjVal["jwt"] = jwtMod
        native.ObjVal["mysql"] = mysqlMod
        native.ObjVal["stripe"] = stripeMod
        native.ObjVal["oauth2"] = oauth2Mod
        native.ObjVal["graphql"] = graphqlMod
        native.ObjVal["rabbitmq"] = rabbitmqMod
        native.ObjVal["excel"] = excelMod
        native.ObjVal["pdf"] = pdfMod
        native.ObjVal["csv"] = csvMod
        native.ObjVal["yaml"] = yamlMod
        native.ObjVal["toml"] = tomlMod
        native.ObjVal["markdown"] = markdownMod
        native.ObjVal["mustache"] = mustacheMod
        native.ObjVal["xml"] = xmlMod
        native.ObjVal["os"] = osMod
        native.ObjVal["alloc"] = allocMod
        native.ObjVal["zip"] = zipMod
        native.ObjVal["regex"] = regexMod
        native.ObjVal["math"] = mathMod
        native.ObjVal["datetime"] = datetimeMod
        native.ObjVal["path"] = pathMod
        native.ObjVal["compress"] = compressMod

        interp.RegisterModule("native", native)
        interp.RegisterModule("io", ioMod)
        interp.RegisterNative("fs", fsMod)
        interp.RegisterNative("http", httpMod)
        interp.RegisterNative("crypto", cryptoMod)
        interp.RegisterNative("db", dbMod)
        interp.RegisterNative("env", envMod)
        interp.RegisterNative("ws", wsMod)
        interp.RegisterNative("mail", mailMod)
        interp.RegisterNative("ai", aiMod)
        interp.RegisterNative("utils", utilsMod)
        interp.RegisterNative("validate", validateMod)
        interp.RegisterNative("postgres", postgresMod)
        interp.RegisterNative("redis", redisMod)
        interp.RegisterNative("jwt", jwtMod)
        interp.RegisterNative("mysql", mysqlMod)
        interp.RegisterNative("stripe", stripeMod)
        interp.RegisterNative("oauth2", oauth2Mod)
        interp.RegisterNative("graphql", graphqlMod)
        interp.RegisterNative("rabbitmq", rabbitmqMod)
        interp.RegisterNative("excel", excelMod)
        interp.RegisterNative("pdf", pdfMod)
        interp.RegisterNative("csv", csvMod)
        interp.RegisterNative("yaml", yamlMod)
        interp.RegisterNative("toml", tomlMod)
        interp.RegisterNative("markdown", markdownMod)
        interp.RegisterNative("mustache", mustacheMod)
        interp.RegisterNative("xml", xmlMod)
        interp.RegisterNative("os", osMod)
        interp.RegisterNative("alloc", allocMod)
        interp.RegisterNative("zip", zipMod)
        interp.RegisterNative("regex", regexMod)
        interp.RegisterNative("math", mathMod)
        interp.RegisterNative("datetime", datetimeMod)
        interp.RegisterNative("path", pathMod)
        interp.RegisterNative("compress", compressMod)
}
