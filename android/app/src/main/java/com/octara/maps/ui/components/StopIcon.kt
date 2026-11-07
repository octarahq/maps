package com.octara.maps.ui.components

import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.AirlineSeatReclineExtra
import androidx.compose.material.icons.filled.DirectionsBus
import androidx.compose.material.icons.filled.LocationOn
import androidx.compose.material.icons.filled.Subway
import androidx.compose.material.icons.filled.Train
import androidx.compose.material.icons.filled.Tram
import androidx.compose.ui.graphics.vector.ImageVector

fun getStopIconResource(stopType: com.octara.maps.types.StopType): ImageVector {
    return when (stopType.toString().uppercase()) {
        "LIFT" -> Icons.Filled.LocationOn
        "METRO" -> Icons.Filled.Train
        "STREET_BUS" -> Icons.Filled.DirectionsBus
        "STREET_TRAM" -> Icons.Filled.Tram
        "RAIL" -> Icons.Filled.Train
        else -> Icons.Filled.LocationOn
    }
}