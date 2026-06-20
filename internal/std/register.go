// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package std

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
	utilsMod := UtilsModule()
	jsonMod := JsonModule()
	jwtMod := JWTModule()
	mathMod := MathModule()
	datetimeMod := DatetimeModule()
	osMod := OsModule()
	regexMod := RegexModule()

	// Build the "native" umbrella object so @import("internal.native") works.
	native := NativeModule()
	native.ObjVal["io"] = ioMod
	native.ObjVal["fs"] = fsMod
	native.ObjVal["http"] = httpMod
	native.ObjVal["crypto"] = cryptoMod
	native.ObjVal["db"] = dbMod
	native.ObjVal["env"] = envMod
	native.ObjVal["ws"] = wsMod
	native.ObjVal["utils"] = utilsMod
	native.ObjVal["json"] = jsonMod
	native.ObjVal["jwt"] = jwtMod
	native.ObjVal["math"] = mathMod
	native.ObjVal["datetime"] = datetimeMod
	native.ObjVal["os"] = osMod
	native.ObjVal["regex"] = regexMod
	interp.RegisterModule("internal.native", native)

	// Register every module directly so @import("fs"), @import("http"), etc.
	// resolve without any .lx shim in between.
	interp.RegisterModule("io", ioMod)
	interp.RegisterModule("fs", fsMod)
	interp.RegisterModule("http", httpMod)
	interp.RegisterModule("crypto", cryptoMod)
	interp.RegisterModule("db", dbMod)
	interp.RegisterModule("env", envMod)
	interp.RegisterModule("ws", wsMod)
	interp.RegisterModule("utils", utilsMod)
	interp.RegisterModule("json", jsonMod)
	interp.RegisterModule("jwt", jwtMod)
	interp.RegisterModule("math", mathMod)
	interp.RegisterModule("datetime", datetimeMod)
	interp.RegisterModule("os", osMod)
	interp.RegisterModule("regex", regexMod)
}
