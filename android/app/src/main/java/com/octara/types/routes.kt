package com.octara.maps.types

import com.google.gson.TypeAdapter
import com.google.gson.annotations.JsonAdapter
import com.google.gson.stream.JsonReader
import com.google.gson.stream.JsonToken
import com.google.gson.stream.JsonWriter

import com.google.gson.annotations.SerializedName

@JsonAdapter(RouteType.Adapter::class)
enum class RouteType(val value: Int) {
    TRAM(0),
    METRO(1),
    TRAIN(2),
    BUS(3),
    FERRY(4),
    CABLE_TRAM(5),
    AERIAL_LIFT(6),
    FUNICULAR(7);

    companion object {
        fun fromValue(value: Int): RouteType =
            entries.firstOrNull { it.value == value } ?: BUS
    }

    class Adapter : TypeAdapter<RouteType>() {
        override fun write(out: JsonWriter, value: RouteType?) {
            out.value(value?.value ?: BUS.value)
        }

        override fun read(reader: JsonReader): RouteType {
            if (reader.peek() == JsonToken.NULL) {
                reader.nextNull()
                return BUS
            }
            return fromValue(reader.nextInt())
        }
    }
}

data class Route(
    val id: String = "",
    @SerializedName("agencyid")
    val agencyId: String = "",
    val short_name: String = "",
    val long_name: String = "",
    val type: RouteType = RouteType.BUS,
    val color: String = "",
    val tcolor: String = "",
    val picto: String? = ""
) {
    val name: String
        get() = if (!short_name.isNullOrEmpty()) short_name else (long_name ?: "")
    val typeName: String
        get() = when (type) {
            RouteType.TRAM -> "Tramway"
            RouteType.METRO -> "Metro"
            RouteType.TRAIN -> "Train"
            RouteType.BUS -> "Bus"
            RouteType.FERRY -> "Ferry"
            RouteType.CABLE_TRAM -> "Cable Tram"
            RouteType.AERIAL_LIFT -> "Aerial Lift"
            RouteType.FUNICULAR -> "Funiculaire"
        }
}