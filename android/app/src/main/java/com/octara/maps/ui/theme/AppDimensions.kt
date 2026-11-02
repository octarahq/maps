package com.octara.maps.ui.theme

import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.ui.unit.dp

object AppDimensions {
    val SearchBarCornerRadius = 24.dp
    val SearchBarShape = RoundedCornerShape(SearchBarCornerRadius)
    val SearchBarHeight = 52.dp
    val SearchBarElevation = 2.dp

    val BottomSheetCornerRadius = 24.dp
    val BottomSheetShape = RoundedCornerShape(topStart = BottomSheetCornerRadius, topEnd = BottomSheetCornerRadius)
    val BottomSheetPeekHeight = 110.dp
    val BottomSheetElevation = 8.dp
}
